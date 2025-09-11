package process

import (
	"container/list"
	"fmt"
	"sort"
	"strings"

	"procmate/pkg/config"
)

// ProcessNode 表示依赖图中的一个进程节点
// 包含进程配置以及其在依赖图中的关系信息
type ProcessNode struct {
	Process      config.Process // 进程配置信息
	Dependencies []*ProcessNode // 该进程依赖的其他进程列表
	Dependents   []*ProcessNode // 依赖于该进程的其他进程列表
	Layer        int            // 该进程在分层执行计划中的层级（0为最底层，无依赖）
}

// DependencyGraph 表示进程依赖关系图
// 提供依赖分析、循环检测、分层规划等核心功能
type DependencyGraph struct {
	nodes  map[string]*ProcessNode // 进程名称到节点的映射
	layers [][]*ProcessNode        // 分层执行计划，每层包含可并行执行的进程名称
}

// buildDependencyGraph 创建一个新的依赖图实例
//
// 参数:
//   - allProcesses: 所有可用进程的映射表，key为进程名称，value为进程配置
//   - requestedServices: 请求启动的进程名称列表
//
// 返回:
//   - *DependencyGraph: 构建好的依赖图
//   - error: 构建过程中遇到的错误（如循环依赖、未定义进程等）
//
// 示例:
//
//	graph, err := buildDependencyGraph(allProcesses, []string{"web", "db"})
//	if err != nil {
//	    log.Fatal("构建依赖图失败:", err)
//	}
func buildDependencyGraph(allProcesses []config.Process, requestedProcesses []config.Process) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		nodes: make(map[string]*ProcessNode),
	}

	// 第一步：构建包含所有必需进程的节点集合
	if err := graph.buildNodes(allProcesses, requestedProcesses); err != nil {
		return nil, fmt.Errorf("构建依赖图节点失败: %w", err)
	}

	// 第二步：建立节点间的双向关系
	if err := graph.buildDependencyRelations(); err != nil {
		return nil, fmt.Errorf("建立依赖关系失败: %w", err)
	}

	// 第三步：检测循环依赖
	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("检测到循环依赖: %w", err)
	}

	// 第四步：计算分层执行计划
	if err := graph.calculateLayers(); err != nil {
		return nil, fmt.Errorf("计算分层执行计划失败: %w", err)
	}

	return graph, nil
}

// buildNodes 递归构建依赖图中的所有节点
// 从请求的服务开始，递归添加所有依赖的进程
func (g *DependencyGraph) buildNodes(allProcesses []config.Process, requestedProcesses []config.Process) error {
	enabledProcessesMap := make(map[string]config.Process)
	for _, p := range allProcesses {
		if p.Enabled {
			enabledProcessesMap[p.Name] = p
		}
	}

	// 使用递归函数来处理每个服务及其依赖
	var addNodeAndGetPointer func(processName string) (*ProcessNode, error)
	addNodeAndGetPointer = func(processName string) (*ProcessNode, error) {
		// a. 如果节点已存在，直接返回它的指针，终止递归
		if existingNode, exists := g.nodes[processName]; exists {
			return existingNode, nil
		}

		// b. 检查进程是否在已启用的配置中定义
		process, exists := enabledProcessesMap[processName]
		if !exists {
			return nil, fmt.Errorf("进程 '%s' 未在配置文件中定义或被禁用", processName)
		}

		// 创建新节点并立即加入到图中，以处理循环依赖
		node := &ProcessNode{
			Process:    process,
			Dependents: []*ProcessNode{}, // 初始化为空切片
		}
		g.nodes[processName] = node

		// 递归处理所有依赖项
		dependencies := make([]*ProcessNode, 0, len(process.DependsOn))
		for _, depName := range process.DependsOn {
			// 递归调用
			depNode, err := addNodeAndGetPointer(depName)
			if err != nil {
				// 包装错误，提供更清晰的上下文
				return nil, fmt.Errorf("处理进程 '%s' 的依赖 '%s' 时失败: %w", processName, depName, err)
			}
			// 将返回的依赖节点指针添加到切片中
			dependencies = append(dependencies, depNode)
		}
		// 将连接好的依赖关系设置回当前节点
		node.Dependencies = dependencies

		// e. 返回创建好的、连接完整的节点指针
		return node, nil
	}

	// 3. 遍历所有请求启动的服务，开始递归构建
	for _, process := range requestedProcesses {
		if _, err := addNodeAndGetPointer(process.Name); err != nil {
		}
	}

	return nil
}

// buildDependencyRelations 建立节点间的双向依赖关系
// 为每个节点填充其依赖者列表，便于后续的层级计算
func (g *DependencyGraph) buildDependencyRelations() error {
	// 遍历图中的每一个节点 (我们称之为 nodeA)
	for _, nodeA := range g.nodes {
		// 遍历 nodeA 的所有依赖项 (我们称之为 nodeB)
		for _, nodeB := range nodeA.Dependencies {
			// 此时，我们已经确定了关系：nodeA 依赖 nodeB (nodeA -> nodeB)

			//  将 nodeA 的指针追加到 nodeB.Dependents
			nodeB.Dependents = append(nodeB.Dependents, nodeA)
		}
	}

	// 使用 sort.Slice 对被依赖者列表进行排序
	for _, node := range g.nodes {
		sort.Slice(node.Dependents, func(i, j int) bool {
			// 根据被依赖者的进程名称进行字母序排序
			return node.Dependents[i].Process.Name < node.Dependents[j].Process.Name
		})
	}

	return nil
}

// detectCycles 使用深度优先搜索检测依赖图中的循环依赖。
// 采用经典的三色标记算法，并显式维护递归栈以准确报告循环路径。
func (g *DependencyGraph) detectCycles() error {
	// 节点状态定义:
	const (
		white = 0 // white: 尚未访问
		gray  = 1 // gray:  正在访问（位于当前递归栈上）
		black = 2 // black: 已完成访问（及其所有后代）
	)

	nodeStates := make(map[string]int, len(g.nodes))
	// 初始化所有节点为“尚未访问”
	for name := range g.nodes {
		nodeStates[name] = white
	}

	// recursionStack 用于追踪当前的递归路径
	var recursionStack []string

	// 深度优先搜索的核心函数
	var dfs func(node *ProcessNode) error
	dfs = func(node *ProcessNode) error {
		nodeName := node.Process.Name

		// a. (标记) 将当前节点标记为“正在访问”，并推入递归栈
		nodeStates[nodeName] = gray
		recursionStack = append(recursionStack, nodeName)

		// b. (探索) 遍历所有依赖项
		for _, depNode := range node.Dependencies {
			depName := depNode.Process.Name

			switch nodeStates[depName] {
			case gray:
				// c. (发现循环) 如果依赖项是灰色，说明我们找到了一个循环。
				//    现在可以根据递归栈构建出准确的循环路径。
				cyclePath := buildCyclePath(recursionStack, depName)
				return fmt.Errorf("发现循环依赖链: %s", strings.Join(cyclePath, " -> "))

			case white:
				// d. (继续深搜) 如果依赖项是白色，对其进行递归访问。
				if err := dfs(depNode); err != nil {
					// 如果下层递归返回了错误，直接将错误向上传递。
					return err
				}
			case black:
				// e. 如果依赖项是黑色，说明它已经被安全地访问完毕，无需任何操作。
			}
		}

		// f. (完成访问) 当前节点的所有后代都已访问完毕，将其标记为“已完成”，并从递归栈中弹出。
		//    使用 defer 可以确保即使在函数提前返回时也能正确执行。
		//    (注意：在这个特定实现中，我们在 return 前手动弹出，defer 仅作为概念展示)
		recursionStack = recursionStack[:len(recursionStack)-1]
		nodeStates[nodeName] = black
		return nil
	}

	// 遍历图中所有节点，作为DFS的起点（以处理非连通图）
	for nodeName, node := range g.nodes {
		if nodeStates[nodeName] == white {
			if err := dfs(node); err != nil {
				// 旦发现循环，立即返回错误
				return err
			}
		}
	}

	return nil
}

// buildCyclePath 是一个辅助函数，用于从递归栈中提取循环路径。
func buildCyclePath(stack []string, cycleTarget string) []string {
	var startIndex int
	for i, name := range stack {
		if name == cycleTarget {
			startIndex = i
			break
		}
	}
	// 循环路径 = 栈中从循环点开始的部分 + 循环点自身（以闭合路径）
	cycle := append(stack[startIndex:], cycleTarget)
	return cycle
}

// calculateLayers 使用高效的 Kahn 算法（基于队列）计算分层执行计划。
func (g *DependencyGraph) calculateLayers() error {
	// 1. 初始化：计算每个节点的初始入度（被依赖次数）
	inDegree := make(map[string]int, len(g.nodes))
	for name, node := range g.nodes {
		inDegree[name] = len(node.Dependencies)
	}

	// 2. 寻找起点：将所有入度为 0 的节点加入队列
	// ✅ 核心修正 1: 使用队列代替循环扫描，提升效率
	queue := list.New()
	for name, degree := range inDegree {
		if degree == 0 {
			queue.PushBack(g.nodes[name]) // ✅ 核心修正 2: 队列中存储 *ProcessNode 指针
		}
	}

	var processedCount int
	// 3. 迭代处理：当队列不为空时，持续处理
	for queue.Len() > 0 {
		currentLayerSize := queue.Len()
		// ✅ 核心修正 3: 当前层的类型是 []*ProcessNode，而不是 []string
		currentLayerNodes := make([]*ProcessNode, 0, currentLayerSize)

		// 处理当前队列中的所有节点，它们构成了新的一层
		for i := 0; i < currentLayerSize; i++ {
			element := queue.Front()
			queue.Remove(element)
			node := element.Value.(*ProcessNode) // 从队列中取出节点

			currentLayerNodes = append(currentLayerNodes, node)
		}

		// 确保输出的确定性
		sort.Slice(currentLayerNodes, func(i, j int) bool {
			return currentLayerNodes[i].Process.Name < currentLayerNodes[j].Process.Name
		})

		// 为当前层的节点设置层级，并更新其下游节点的入度
		for _, node := range currentLayerNodes {
			node.Layer = len(g.layers) // 设置层级
			processedCount++

			// 减少所有依赖于当前节点的“下游”节点的入度
			for _, dependentNode := range node.Dependents {
				dependentName := dependentNode.Process.Name
				inDegree[dependentName]--
				// 如果下游节点的入度变为 0，则将其加入队列，备战下一层
				if inDegree[dependentName] == 0 {
					queue.PushBack(dependentNode)
				}
			}
		}

		g.layers = append(g.layers, currentLayerNodes)
	}

	// 4. 验证：如果处理过的节点数少于总节点数，说明图中存在循环
	if processedCount < len(g.nodes) {
		var remainingNodes []string
		for name, degree := range inDegree {
			if degree > 0 { // 入度大于0的节点就是循环的一部分
				remainingNodes = append(remainingNodes, name)
			}
		}
		sort.Strings(remainingNodes)
		return fmt.Errorf("无法完成分层，检测到循环依赖，涉及的进程有: %v", remainingNodes)
	}

	return nil
}

// GetExecutionLayers 获取分层执行计划
// 返回二维数组，每个子数组包含一层中可并行执行的进程配置
//
// 返回:
//   - [][]config.Process: 分层执行计划，外层数组表示执行顺序，内层数组表示并行执行的进程
func (g *DependencyGraph) GetExecutionLayers() [][]config.Process {
	result := make([][]config.Process, len(g.layers))

	for i, layer := range g.layers {
		result[i] = make([]config.Process, len(layer))
		for j, node := range layer {
			// ✅ 核心职责：提取 node 指针中的 Process 值
			result[i][j] = node.Process
		}
	}

	return result
}

// GetExecutionLayers 全局函数，提供新的分层执行计划功能
// 这是新增的API，用于支持并行启动
//
// 参数:
//   - allProcesses: 所有进程配置的映射表
//   - requestedServices: 请求启动的进程名称列表
//
// 返回:
//   - [][]config.Process: 分层执行计划
//   - error: 处理过程中的错误
func GetExecutionLayers(allProcesses []config.Process, requestedProcesses []config.Process) ([][]config.Process, error) {
	graph, err := buildDependencyGraph(allProcesses, requestedProcesses)
	if err != nil {
		return nil, err
	}

	return graph.GetExecutionLayers(), nil
}

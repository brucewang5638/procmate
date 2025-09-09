package process

import (
	"fmt"
	"sort"
	"strings"

	"procmate/pkg/config"
)

// ProcessNode 表示依赖图中的一个进程节点
// 包含进程配置以及其在依赖图中的关系信息
type ProcessNode struct {
	Process      config.Process // 进程配置信息
	Dependencies []string       // 该进程依赖的其他进程名称列表
	Dependents   []string       // 依赖于该进程的其他进程名称列表
	Layer        int            // 该进程在分层执行计划中的层级（0为最底层，无依赖）
}

// DependencyGraph 表示进程依赖关系图
// 提供依赖分析、循环检测、分层规划等核心功能
type DependencyGraph struct {
	nodes map[string]*ProcessNode // 进程名称到节点的映射
	layers [][]string              // 分层执行计划，每层包含可并行执行的进程名称
}

// NewDependencyGraph 创建一个新的依赖图实例
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
//   graph, err := NewDependencyGraph(allProcesses, []string{"web", "db"})
//   if err != nil {
//       log.Fatal("构建依赖图失败:", err)
//   }
func NewDependencyGraph(allProcesses map[string]config.Process, requestedServices []string) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		nodes: make(map[string]*ProcessNode),
	}

	// 第一步：构建包含所有必需进程的节点集合
	if err := graph.buildNodes(allProcesses, requestedServices); err != nil {
		return nil, fmt.Errorf("构建依赖图节点失败: %w", err)
	}

	// 第二步：建立节点间的依赖关系
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
func (g *DependencyGraph) buildNodes(allProcesses map[string]config.Process, services []string) error {
	// 使用递归函数来处理每个服务及其依赖
	var addNode func(serviceName string) error
	addNode = func(serviceName string) error {
		// 如果节点已存在，跳过处理
		if _, exists := g.nodes[serviceName]; exists {
			return nil
		}

		// 检查进程是否在配置中定义
		process, exists := allProcesses[serviceName]
		if !exists {
			return fmt.Errorf("进程 '%s' 未在配置文件中定义", serviceName)
		}

		// 检查进程是否已启用
		if !process.Enabled {
			return fmt.Errorf("进程 '%s' 已被禁用 (enabled: false)", serviceName)
		}

		// 创建新节点
		node := &ProcessNode{
			Process:      process,
			Dependencies: make([]string, len(process.DependsOn)),
			Dependents:   []string{},
		}
		copy(node.Dependencies, process.DependsOn)

		// 将节点添加到图中
		g.nodes[serviceName] = node

		// 递归处理所有依赖项
		for _, depName := range process.DependsOn {
			if err := addNode(depName); err != nil {
				return fmt.Errorf("处理进程 '%s' 的依赖 '%s' 时失败: %w", serviceName, depName, err)
			}
		}

		return nil
	}

	// 处理所有请求的服务
	for _, serviceName := range services {
		if err := addNode(serviceName); err != nil {
			return err
		}
	}

	return nil
}

// buildDependencyRelations 建立节点间的双向依赖关系
// 为每个节点填充其依赖者列表，便于后续的层级计算
func (g *DependencyGraph) buildDependencyRelations() error {
	for nodeName, node := range g.nodes {
		for _, depName := range node.Dependencies {
			// 验证依赖的进程是否存在于图中
			depNode, exists := g.nodes[depName]
			if !exists {
				return fmt.Errorf("进程 '%s' 的依赖 '%s' 不存在于依赖图中", nodeName, depName)
			}

			// 将当前进程添加到其依赖进程的依赖者列表中
			depNode.Dependents = append(depNode.Dependents, nodeName)
		}
	}

	// 对依赖者列表进行排序，确保输出的确定性
	for _, node := range g.nodes {
		sort.Strings(node.Dependents)
	}

	return nil
}

// detectCycles 使用深度优先搜索检测依赖图中的循环依赖
// 采用经典的白灰黑三色标记算法
func (g *DependencyGraph) detectCycles() error {
	// 节点状态定义：
	// white (0): 未访问
	// gray (1):  正在访问中（在当前DFS路径上）
	// black (2): 已访问完成
	const (
		white = 0
		gray  = 1
		black = 2
	)

	nodeStates := make(map[string]int)
	var cycleNodes []string

	// 深度优先搜索函数
	var dfs func(nodeName string) error
	dfs = func(nodeName string) error {
		// 将当前节点标记为正在访问
		nodeStates[nodeName] = gray

		node := g.nodes[nodeName]
		for _, depName := range node.Dependencies {
			switch nodeStates[depName] {
			case gray:
				// 发现回边，存在循环依赖
				cycleNodes = append(cycleNodes, depName)
				return fmt.Errorf("检测到循环依赖")
			case white:
				// 继续深度搜索未访问的节点
				if err := dfs(depName); err != nil {
					cycleNodes = append(cycleNodes, depName)
					return err
				}
			}
		}

		// 当前节点访问完成
		nodeStates[nodeName] = black
		return nil
	}

	// 对所有节点执行DFS检查
	for nodeName := range g.nodes {
		if nodeStates[nodeName] == white {
			if err := dfs(nodeName); err != nil {
				// 反转循环路径以显示正确的依赖顺序
				for i, j := 0, len(cycleNodes)-1; i < j; i, j = i+1, j-1 {
					cycleNodes[i], cycleNodes[j] = cycleNodes[j], cycleNodes[i]
				}
				return fmt.Errorf("发现循环依赖链: %s", strings.Join(cycleNodes, " -> "))
			}
		}
	}

	return nil
}

// calculateLayers 计算分层执行计划
// 使用修改的Kahn算法，将进程按依赖关系分层
// 同层内的进程可以并行执行，层与层之间必须串行执行
func (g *DependencyGraph) calculateLayers() error {
	// 计算每个节点的入度（依赖数量）
	inDegree := make(map[string]int)
	for nodeName, node := range g.nodes {
		inDegree[nodeName] = len(node.Dependencies)
	}

	currentLayer := 0

	// 持续处理直到所有节点都被分层
	for len(inDegree) > 0 {
		// 查找当前层中入度为0的节点（无未满足依赖）
		var currentLayerNodes []string
		for nodeName, degree := range inDegree {
			if degree == 0 {
				currentLayerNodes = append(currentLayerNodes, nodeName)
			}
		}

		// 如果没有入度为0的节点但还有未处理的节点，说明存在循环依赖
		if len(currentLayerNodes) == 0 {
			var remainingNodes []string
			for nodeName := range inDegree {
				remainingNodes = append(remainingNodes, nodeName)
			}
			return fmt.Errorf("无法完成分层，可能存在未检测到的循环依赖，剩余节点: %v", remainingNodes)
		}

		// 对当前层的节点进行排序，确保输出的确定性
		sort.Strings(currentLayerNodes)

		// 为当前层的节点设置层级信息
		for _, nodeName := range currentLayerNodes {
			g.nodes[nodeName].Layer = currentLayer
			delete(inDegree, nodeName)

			// 更新依赖于当前节点的其他节点的入度
			for _, dependent := range g.nodes[nodeName].Dependents {
				if _, exists := inDegree[dependent]; exists {
					inDegree[dependent]--
				}
			}
		}

		// 将当前层添加到分层计划中
		g.layers = append(g.layers, currentLayerNodes)
		currentLayer++
	}

	return nil
}

// GetExecutionLayers 获取分层执行计划
// 返回二维数组，每个子数组包含一层中可并行执行的进程配置
//
// 返回:
//   - [][]config.Process: 分层执行计划，外层数组表示执行顺序，内层数组表示并行执行的进程
//
// 示例:
//   layers := graph.GetExecutionLayers()
//   for i, layer := range layers {
//       fmt.Printf("第%d层 (并行执行): ", i+1)
//       for _, process := range layer {
//           fmt.Printf("%s ", process.Name)
//       }
//       fmt.Println()
//   }
func (g *DependencyGraph) GetExecutionLayers() [][]config.Process {
	result := make([][]config.Process, len(g.layers))
	
	for i, layer := range g.layers {
		result[i] = make([]config.Process, len(layer))
		for j, nodeName := range layer {
			result[i][j] = g.nodes[nodeName].Process
		}
	}
	
	return result
}

// GetExecutionPlan 获取线性执行计划（保持向后兼容性）
// 将分层计划展平为线性顺序，用于需要串行执行的场景
//
// 返回:
//   - []config.Process: 按依赖关系排序的进程配置列表
func (g *DependencyGraph) GetExecutionPlan() []config.Process {
	var result []config.Process
	
	for _, layer := range g.layers {
		for _, nodeName := range layer {
			result = append(result, g.nodes[nodeName].Process)
		}
	}
	
	return result
}

// GetNodeInfo 获取指定进程的节点信息
// 用于调试和状态查询
//
// 参数:
//   - processName: 进程名称
//
// 返回:
//   - *ProcessNode: 进程节点信息，如果进程不存在则返回nil
func (g *DependencyGraph) GetNodeInfo(processName string) *ProcessNode {
	return g.nodes[processName]
}

// GetStats 获取依赖图统计信息
// 返回依赖图的基本统计数据，用于调试和监控
//
// 返回:
//   - map[string]any: 包含统计信息的映射表
func (g *DependencyGraph) GetStats() map[string]any {
	stats := map[string]any{
		"total_processes": len(g.nodes),
		"total_layers":    len(g.layers),
		"layers_detail":   make([]map[string]any, len(g.layers)),
	}

	for i, layer := range g.layers {
		stats["layers_detail"].([]map[string]any)[i] = map[string]any{
			"layer_index":     i,
			"processes_count": len(layer),
			"processes":       layer,
		}
	}

	return stats
}

// 兼容性函数：保持与现有代码的兼容性

// GetExecutionPlan 全局函数，保持向后兼容性
// 这是原有API的兼容版本，内部使用新的依赖图实现
//
// 参数:
//   - allProcesses: 所有进程配置的映射表
//   - requestedServices: 请求启动的进程名称列表
//
// 返回:
//   - []config.Process: 按依赖关系排序的进程配置列表
//   - error: 处理过程中的错误
func GetExecutionPlan(allProcesses map[string]config.Process, requestedServices []string) ([]config.Process, error) {
	graph, err := NewDependencyGraph(allProcesses, requestedServices)
	if err != nil {
		return nil, err
	}
	
	return graph.GetExecutionPlan(), nil
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
func GetExecutionLayers(allProcesses map[string]config.Process, requestedServices []string) ([][]config.Process, error) {
	graph, err := NewDependencyGraph(allProcesses, requestedServices)
	if err != nil {
		return nil, err
	}
	
	return graph.GetExecutionLayers(), nil
}
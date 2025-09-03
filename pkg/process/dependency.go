package process

import (
	"fmt"
	"sort"
	"strings"

	"procmate/pkg/config"
)

// 确定给定服务集的正确启动顺序，包括它们的依赖关系。它对依赖关系图执行拓扑排序。
// 它按照应执行的顺序返回服务，如果检测到循环依赖关系或未定义依赖关系。
func GetExecutionPlan(allProcesses map[string]config.Process, requestedServices []string) ([]config.Process, error) {
	// 1. 构建需要启动的所有服务集，包括依赖项。
	planSet, err := buildPlanSet(allProcesses, requestedServices)
	if err != nil {
		return nil, err
	}

	// 2. 对计划中的服务执行拓扑排序。
	sortedNames, err := topologicalSort(allProcesses, planSet)
	if err != nil {
		return nil, err
	}

	// 3. 将排序后的名称转换回服务对象。
	result := make([]config.Process, len(sortedNames))
	for i, name := range sortedNames {
		result[i] = allProcesses[name]
	}

	return result, nil
}

// 递归查找初始请求所需的所有服务.
func buildPlanSet(allProcesses map[string]config.Process, requested []string) (map[string]bool, error) {
	plan := make(map[string]bool)
	var visit func(name string) error

	visit = func(name string) error {
		if plan[name] {
			return nil // Already visited
		}

		process, exists := allProcesses[name]
		if !exists {
			return fmt.Errorf("service '%s' is not defined in config.yaml", name)
		}

		plan[name] = true
		for _, depName := range process.DependsOn {
			if err := visit(depName); err != nil {
				return fmt.Errorf("failed to resolve dependency for '%s': %w", name, err)
			}
		}
		return nil
	}

	for _, serviceName := range requested {
		if err := visit(serviceName); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

// topologicalSort 根据给定的服务集的依赖关系对它们进行排序。
func topologicalSort(allProcesses map[string]config.Process, planSet map[string]bool) ([]string, error) {
	inDegree := make(map[string]int)
	graph := make(map[string][]string) // dependency -> dependents

	for name := range planSet {
		inDegree[name] = 0
		graph[name] = []string{}
	}

	for name := range planSet {
		process := allProcesses[name]
		for _, depName := range process.DependsOn {
			// Ensure the dependency is also in the plan. buildPlanSet should guarantee this.
			if _, ok := planSet[depName]; ok {
				graph[depName] = append(graph[depName], name)
				inDegree[name]++
			}
		}
	}

	queue := []string{}
	for name := range planSet {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}
	// 对初始队列（无依赖的服务）进行排序，以保证一个可预测的、确定的启动顺序。
	sort.Strings(queue)

	sorted := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// 服务在依赖满足后以更自然的顺序入队。
		dependents := graph[current]

		for _, dependent := range dependents {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(planSet) {
		var cycleParticipants []string
		for name, degree := range inDegree {
			if degree > 0 {
				cycleParticipants = append(cycleParticipants, name)
			}
		}
		sort.Strings(cycleParticipants)
		return nil, fmt.Errorf("circular dependency detected involving services: %s", strings.Join(cycleParticipants, ", "))
	}

	return sorted, nil
}

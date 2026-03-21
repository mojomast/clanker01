package task

type DecomposeStrategy string

const (
	StrategySequential DecomposeStrategy = "sequential"
	StrategyParallel   DecomposeStrategy = "parallel"
	StrategyPipeline   DecomposeStrategy = "pipeline"
	StrategyMapReduce  DecomposeStrategy = "map_reduce"
	StrategyDivide     DecomposeStrategy = "divide_conquer"
)

type DecompositionRule struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Condition Condition       `json:"condition"`
	Action    DecomposeAction `json:"action"`
	Priority  int             `json:"priority"`
}

type Condition struct {
	TaskKind      TaskKind `json:"task_kind"`
	MinComplexity float64  `json:"min_complexity"`
	HasPattern    string   `json:"has_pattern"`
	InputSize     int      `json:"min_input_size"`
}

type DecomposeAction struct {
	Strategy  DecomposeStrategy `json:"strategy"`
	Template  string            `json:"template"`
	ChunkSize int               `json:"chunk_size,omitempty"`
}

var DefaultRules = []DecompositionRule{
	{
		ID:   "large-compute-parallel",
		Name: "Parallelize Large Compute Tasks",
		Condition: Condition{
			TaskKind:      KindCompute,
			MinComplexity: 0.7,
			InputSize:     1000,
		},
		Action: DecomposeAction{
			Strategy:  StrategyParallel,
			ChunkSize: 100,
		},
		Priority: 10,
	},
	{
		ID:   "pipeline-io-network",
		Name: "Pipeline IO and Network",
		Condition: Condition{
			TaskKind:   KindIO,
			HasPattern: "network_required",
		},
		Action: DecomposeAction{
			Strategy: StrategyPipeline,
		},
		Priority: 5,
	},
	{
		ID:   "sequential-dependencies",
		Name: "Sequential for Dependencies",
		Condition: Condition{
			HasPattern: "strict_order",
		},
		Action: DecomposeAction{
			Strategy: StrategySequential,
		},
		Priority: 15,
	},
	{
		ID:   "mapreduce-aggregate",
		Name: "MapReduce for Aggregate Tasks",
		Condition: Condition{
			TaskKind:      KindAggregate,
			MinComplexity: 0.6,
			InputSize:     500,
		},
		Action: DecomposeAction{
			Strategy:  StrategyMapReduce,
			ChunkSize: 50,
		},
		Priority: 8,
	},
}

type ComplexityAnalyzer interface {
	Analyze(task *Task) float64
}

type DefaultComplexityAnalyzer struct{}

func (d *DefaultComplexityAnalyzer) Analyze(task *Task) float64 {
	if task.Input == nil {
		return 0.1
	}

	inputSize := len(task.Input)
	complexity := 0.1 + float64(inputSize)*0.01

	if items, ok := task.Input["items"].([]any); ok {
		complexity += float64(len(items)) * 0.05
	}

	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

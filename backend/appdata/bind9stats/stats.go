package bind9stats

// A structure holding named zone statistics.
type Bind9StatsZone struct {
	Name     string
	Class    string
	Serial   uint32
	ZoneType string
}

// A structure holding named resolver statistics.
type Bind9StatsResolver struct {
	Stats      map[string]int64
	Qtypes     map[string]int64
	Cache      map[string]int64
	CacheStats map[string]int64
	Adb        map[string]int64
}

// A structure holding named view statistics.
type Bind9StatsView struct {
	Zones    []*Bind9StatsZone
	Resolver *Bind9StatsResolver
}

// A structure holding named socket statistics.
type Bind9StatsSocket struct {
	ID           string
	References   int64
	SocketType   string
	PeerAddress  string
	LocalAddress string
	States       []string
}

// A structure holding named socket manager statistics.
type Bind9StatsSocketMgr struct {
	Sockets []*Bind9StatsSocket
}

// A structure holding named task statistics.
type Bind9StatsTask struct {
	ID         string
	Name       string
	References int64
	State      string
	Quantum    int64
	Events     int64
}

// A structure holding named task manager statistics.
type Bind9StatsTaskMgr struct {
	ThreadModel    string
	WorkerThreads  int64
	DefaultQuantum int64
	TasksRunning   int64
	TasksReady     int64
	Tasks          []*Bind9StatsTask
}

// A structure holding named context statistics.
type Bind9StatsContext struct {
	ID         string
	Name       string
	References int64
	Total      int64
	InUse      int64
	MaxInUse   int64
	BlockSize  int64
	Pools      int64
	HiWater    int64
	LoWater    int64
}

// A structure holding named memory statistics.
type Bind9StatsMemory struct {
	TotalUse    int64
	InUse       int64
	BlockSize   int64
	ContextSize int64
	Lost        int64
	Contexts    []*Bind9StatsContext
}

// A structure holding named traffic statistics.
type Bind9StatsTraffic struct {
	SizeBucket map[string]int64
}

// A structure holding named statistics.
type Bind9NamedStats struct {
	JSONStatsVersion string
	BootTime         string
	ConfigTime       string
	CurrentTime      string
	NamedVersion     string
	OpCodes          map[string]int64
	Rcodes           map[string]int64
	Qtypes           map[string]int64
	NsStats          map[string]int64
	Views            map[string]*Bind9StatsView
	SockStats        map[string]int64
	SocketMgr        *Bind9StatsSocketMgr
	TaskMgr          *Bind9StatsTaskMgr
	Memory           *Bind9StatsMemory
	Traffic          map[string]*Bind9StatsTraffic
}

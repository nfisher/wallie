package wallie

type Config struct {
	JiraBase         string
	LoginPath        string
	SessionName      string
	AlwaysReloadHTML bool `json:"-"`
	IsInsecure       bool
}

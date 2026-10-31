package command

type Type string

const (
	TypeExec       Type = "exec"
	TypePickFolder Type = "pick_folder"
	TypePickFile   Type = "pick_file"
	TypeOpenPath   Type = "open_path"
	TypeURL        Type = "url"
)

type Definition struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description,omitempty"`
	Type        Type     `toml:"type"`
	Executable  string   `toml:"executable,omitempty"`
	Arguments   []string `toml:"args,omitempty"`
	WorkingDir  string   `toml:"working_dir,omitempty"`
	Path        string   `toml:"path,omitempty"`
	URL         string   `toml:"url,omitempty"`
	Hotkey      string   `toml:"hotkey,omitempty"`
	StartDir    string   `toml:"start_dir,omitempty"`
}

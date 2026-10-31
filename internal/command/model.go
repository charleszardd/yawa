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
	Name        string   `toml:"name" json:"name"`
	Description string   `toml:"description,omitempty" json:"description"`
	Type        Type     `toml:"type" json:"type"`
	Executable  string   `toml:"executable,omitempty" json:"executable"`
	Arguments   []string `toml:"args,omitempty" json:"args"`
	WorkingDir  string   `toml:"working_dir,omitempty" json:"working_dir"`
	Path        string   `toml:"path,omitempty" json:"path"`
	URL         string   `toml:"url,omitempty" json:"url"`
	Hotkey      string   `toml:"hotkey,omitempty" json:"hotkey"`
	StartDir    string   `toml:"start_dir,omitempty" json:"start_dir"`
}

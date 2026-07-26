package pages

// AboutData holds version and build metadata for the About page.
type AboutData struct {
	Version    string
	Buildstamp string
	GoVersion  string
	Platform   string
}

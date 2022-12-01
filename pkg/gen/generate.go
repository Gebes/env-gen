package gen

import (
	"fmt"
	"github.com/joho/godotenv"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var titleFormatter = cases.Title(language.English)

type Field struct {
	EnvName  string
	CodeName string
	Type     string
}
type Data struct {
	Fields []Field
	Config Config
}

func Generate(config Config) error {
	envMap, err := godotenv.Read(config.Env)
	if err != nil {
		return fmt.Errorf("could not parse .env file: %w", err)
	}
	/* better way
	// TODO: find better way to embed template
	tmpl, err := template.ParseFiles("./pkg/gen/templates/env.tmpl")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}*/
	tmpl, err := template.New("env").Parse(`
{{/* The line below tells Intellij/GoLand to enable the autocompletion based on the *gen.Data type. */}}
{{/* gotype: github.com/Gebes/env-gen/pkg/gen.Data */}}

package {{ $.Config.PackageName }}

import (
   {{ if $.Config.GodotEnvEnabled}} "github.com/joho/godotenv"{{end}}
    "log"
    "os"
    "strconv"
)

var (
{{- range $f := $.Fields }}
    {{ $f.CodeName }} {{ $f.Type }}
{{- end }}
)

func init() {
    {{ if $.Config.GodotEnvEnabled}}{{ if $.Config.GodotEnvLoggingEnabled}}
	err := godotenv.Load()
    if err != nil {
        log.Println("Could not load .env file:", err)
    }{{end}}{{ if not $.Config.GodotEnvLoggingEnabled}}
    _ = godotenv.Load()
    {{end}}
    {{end}}

    {{ $needsError := (or $.Config.ExitOnParseError $.Config.LogParseError) }}
    {{ $log := $.Config.LogParseError }}
	{{ if and (or (not $.Config.GodotEnvEnabled) (not $.Config.GodotEnvLoggingEnabled)) $needsError }}
	var err error{{end}}

    {{ if $.Config.ExitOnParseError }}hasError := false{{ end }}

    {{- range $f := $.Fields }}{{ if eq $f.Type "string"}}
    {{ $f.CodeName }} = os.Getenv("{{ $f.EnvName }}"){{ end }}{{ if eq $f.Type "int" }}
    {{ $f.CodeName }}, {{ if $needsError }}err{{end}}{{if not $needsError}}_{{end}} = strconv.Atoi(os.Getenv("{{ $f.EnvName }}"))
    {{if $needsError}}if err != nil {
        {{ if $.Config.ExitOnParseError }}hasError = true
        {{ end }}{{if $log }}log.Println("Could not parse variable {{$f.EnvName}} to a int from the environment:", err){{end}}
    }{{ end }}{{end}}{{  if eq $f.Type "bool" }}
    {{ $f.CodeName }}, {{ if $needsError }}err{{end}}{{if not $needsError}}_{{end}} = strconv.ParseBool(os.Getenv("{{ $f.EnvName }}"))
    {{if $needsError}}if err != nil {
        {{ if $.Config.ExitOnParseError }}hasError = true
        {{ end }}{{if $log }}log.Println("Could not parse variable {{$f.EnvName}} to a bool from the environment:", err){{end}}
    }{{ end }}{{end}}
    {{- end }}
    {{ if $.Config.ExitOnParseError }}
    if hasError {
        os.Exit(1)
    }{{ end }}
}
`)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	var fields []Field
	for key, value := range envMap {
		field := Field{
			EnvName:  key,
			CodeName: toVariableName(key),
			Type:     "string",
		}

		switch {
		case isInt(value):
			field.Type = "int"
			break
		case isBool(value):
			field.Type = "bool"
			break
		}

		fields = append(fields, field)
	}

	sort.Slice(fields, func(i, j int) bool {
		return strings.Compare(fields[i].CodeName, fields[j].CodeName) < 0
	})

	file, err := os.Create(config.Output)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	err = tmpl.Execute(file, Data{
		fields,
		config,
	})
	if err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("file close: %w", err)
	}

	return nil
}

func toVariableName(envKey string) string {
	split := strings.Split(envKey, "_")
	for i := range split {
		split[i] = titleFormatter.String(split[i])
	}
	return strings.Join(split, "")
}

func isInt(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func isBool(value string) bool {
	_, err := strconv.ParseBool(value)
	return err == nil
}

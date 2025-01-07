package main

import (
	"flag"
	"os"
)

// isFlagPassed checks if a flag has been set by the user.
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// mergeWithEnv merges the flags with the environment variables. It uses the flag name as the environment variable name.
// If envPrefix is set, it will use the prefix and the flag name separated by an underscore.
// It will only set the flag if it has not been set by the user.
// If the flag has a _FILE suffix, it will read the value from the file.
func mergeWithEnv(fs *flag.FlagSet, envPrefix string) error {
	fs.VisitAll(func(f *flag.Flag) {
		if !isFlagPassed(f.Name) {
			if val, ok := lookupVar(f.Name, envPrefix); ok {
				fs.Set(f.Name, val)
			}
		}
	})
	return nil
}

// lookupEnv looks up the environment variable by name.
// If envPrefix is set, it will use the prefix and the name separated by an underscore.
func lookupEnv(name, envPrefix string) (string, bool) {
	if envPrefix != "" {
		name = envPrefix + "_" + name
	}
	val := os.Getenv(name)

	return val, val != ""
}

func readVarFromFile(file string) (string, bool) {
	if s, err := os.Stat(file); err == nil && !s.IsDir() {
		if b, err := os.ReadFile(file); err == nil {
			return string(b), true
		}
	}
	return "", false
}

func lookupVar(name, envPrefix string) (string, bool) {
	val, ok := lookupEnv(name, envPrefix)
	if !ok {
		if val, ok = lookupEnv(name+"_FILE", ""); ok {
			val, ok = readVarFromFile(val)
		}
	}
	return val, ok
}

package goflux

import (
	"fmt"
	"reflect"
)

// DIError represents different types of dependency injection errors
type DIError struct {
	Type      string
	Operation string
	File      string
	Line      int
	Details   interface{}
}

// MissingDependencies contains details about missing dependencies
type MissingDependencies struct {
	MissingTypes  []reflect.Type
	AvailableDeps map[reflect.Type]*Dependency
}

// DuplicateDependencies contains details about duplicate dependencies
type DuplicateDependencies struct {
	ConflictingType reflect.Type
	ExistingDep     *Dependency
	NewDep          *Dependency
}

// FormatMissingDependenciesError formats and logs a missing dependencies error
func FormatMissingDependenciesError(operation, file string, line int, details MissingDependencies) {
	fmt.Printf("\x1b[31mERROR: \x1b[0m\x1b[1m%d missing dependencies in operation '\x1b[38;5;39m%s\x1b[0m\x1b[1m':\x1b[0m\n",
		len(details.MissingTypes), operation)
	fmt.Printf("\x1b[38;5;45m   Location: \x1b[38;5;255m%s:%d\x1b[0m\n", file, line)

	fmt.Printf("\x1b[31m   Missing dependencies:\x1b[0m\n")
	for i, missingType := range details.MissingTypes {
		fmt.Printf("\x1b[38;5;203m   - Parameter %d: \x1b[38;5;201m%v\x1b[0m\n", i, missingType)
	}

	if len(details.AvailableDeps) > 0 {
		fmt.Printf("\x1b[33m   Available dependencies:\x1b[0m\n")
		for depType, dep := range details.AvailableDeps {
			fmt.Printf("\x1b[38;5;118m   - '\x1b[38;5;226m%s\x1b[38;5;118m' (type: \x1b[38;5;201m%v\x1b[38;5;118m)\x1b[0m\n", dep.Name(), depType)
		}
	} else {
		fmt.Printf("\x1b[33m   No dependencies are currently registered for this procedure.\x1b[0m\n")
	}

	fmt.Printf("\x1b[38;5;118m   Solutions:\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Add the missing dependencies to your procedure using .Inject()\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Remove the unused parameters from your handler function\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Create wrapper types if you have type conflicts\x1b[0m\n")
	fmt.Println() // Add spacing before panic
}

// FormatDuplicateDependenciesError formats and logs a duplicate dependencies error
func FormatDuplicateDependenciesError(operation, file string, line int, details DuplicateDependencies) {
	fmt.Printf("\x1b[31mERROR: \x1b[0m\x1b[1mDuplicate dependency types in operation '\x1b[38;5;39m%s\x1b[0m\x1b[1m':\x1b[0m\n", operation)
	fmt.Printf("\x1b[38;5;45m   Location: \x1b[38;5;255m%s:%d\x1b[0m\n", file, line)

	fmt.Printf("\x1b[31m   Conflicting dependencies:\x1b[0m\n")
	fmt.Printf("\x1b[38;5;203m   - '\x1b[38;5;226m%s\x1b[38;5;203m' (type: \x1b[38;5;201m%v\x1b[38;5;203m)\x1b[0m\n", details.ExistingDep.Name(), details.ConflictingType)
	fmt.Printf("\x1b[38;5;203m   - '\x1b[38;5;226m%s\x1b[38;5;203m' (type: \x1b[38;5;201m%v\x1b[38;5;203m)\x1b[0m\n", details.NewDep.Name(), details.ConflictingType)

	fmt.Printf("\x1b[38;5;118m   Solutions:\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Create wrapper types to disambiguate (e.g., AdminUser, RegularUser)\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Use only one dependency of this type\x1b[0m\n")
	fmt.Printf("\x1b[38;5;118m   • Rename one of the dependencies to provide a different type\x1b[0m\n")
	fmt.Println() // Add spacing before panic
}

// FormatUnusedDependenciesWarning formats and logs unused dependencies warning
func FormatUnusedDependenciesWarning(operation, file string, line int, unusedDeps []*Dependency) {
	fmt.Printf("\x1b[38;5;208mWARNING: \x1b[0m\x1b[1m%d unused dependencies in operation '\x1b[38;5;39m%s\x1b[0m\x1b[1m':\x1b[0m\n",
		len(unusedDeps), operation)
	fmt.Printf("\x1b[38;5;45m   Location: \x1b[38;5;255m%s:%d\x1b[0m\n", file, line)

	for _, dep := range unusedDeps {
		fmt.Printf("\x1b[38;5;203m   - '\x1b[38;5;226m%s\x1b[38;5;203m' (type: \x1b[38;5;201m%v\x1b[38;5;203m) - consider removing from procedure or use it as a dependency\x1b[0m\n",
			dep.Name(), dep.Type())
	}
	fmt.Printf("\x1b[38;5;118m   Tip: Remove unused dependencies to improve performance or use them as dependencies\x1b[0m\n")
	fmt.Println() // Add spacing after warnings
}

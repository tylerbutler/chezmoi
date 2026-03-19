package chezmoi

import (
	"reflect"
	"text/template"

	"github.com/nikolalohinski/gonja/v2/exec"
)

// WrapAsGonjaGlobals converts a template.FuncMap into a map[string]any
// suitable for inclusion in a gonja exec.Context. gonja can call any
// Go function placed in the context via reflection.
func WrapAsGonjaGlobals(funcs template.FuncMap) map[string]any {
	globals := make(map[string]any, len(funcs))
	for name, fn := range funcs {
		globals[name] = fn
	}
	return globals
}

// WrapAsGonjaFilters converts template.FuncMap functions with a "transform"
// signature (first parameter is the piped input) into gonja FilterFunctions.
// Only functions with at least one parameter are included.
func WrapAsGonjaFilters(funcs template.FuncMap) map[string]exec.FilterFunction {
	filters := make(map[string]exec.FilterFunction)
	for name, fn := range funcs {
		fnVal := reflect.ValueOf(fn)
		fnType := fnVal.Type()
		if fnType.NumIn() < 1 {
			continue
		}
		name, fnVal, fnType := name, fnVal, fnType
		filters[name] = func(e *exec.Evaluator, in *exec.Value, params *exec.VarArgs) *exec.Value {
			args := make([]reflect.Value, fnType.NumIn())
			args[0] = convertToReflectValue(in, fnType.In(0))
			for i := 1; i < fnType.NumIn(); i++ {
				if i-1 < len(params.Args) {
					args[i] = convertToReflectValue(params.Args[i-1], fnType.In(i))
				} else {
					args[i] = reflect.Zero(fnType.In(i))
				}
			}
			results := fnVal.Call(args)
			if len(results) == 0 {
				return exec.AsValue(nil)
			}
			if fnType.NumOut() == 2 && fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !results[1].IsNil() {
					return exec.AsValue(results[1].Interface().(error)) //nolint:forcetypeassert
				}
			}
			return exec.AsValue(results[0].Interface())
		}
	}
	return filters
}

func convertToReflectValue(v *exec.Value, targetType reflect.Type) reflect.Value {
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(v.String())
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return reflect.ValueOf(v.Integer()).Convert(targetType)
	case reflect.Float64, reflect.Float32:
		return reflect.ValueOf(v.Float()).Convert(targetType)
	case reflect.Bool:
		return reflect.ValueOf(v.Bool())
	default:
		return reflect.ValueOf(v.Interface())
	}
}

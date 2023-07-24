package exps

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"

	// dot imports only used in parseNumber below
	. "reflect"
	"regexp"
	"strconv"
	"strings"
)

const TagName = "vals"

type fieldInfo struct {
	Name string
	Vals []interface{}
}

func Must[T any](v T, e error) T {
	if e != nil {
		panic(fmt.Sprintf("must: unexpected error: %v", e))
	}
	return v
}

func parseNumber(v string, k reflect.Kind) interface{} {
	var res interface{}
	var err error
	// not handling 64 bit numbers for now.
	// is that even needed?
	switch k {
	case Int, Int8, Int16, Int32:
		var tmp int64
		tmp, err = strconv.ParseInt(v, 0, 64)
		res = int(tmp)
	case Uint, Uint8, Uint16, Uint32:
		res, err = strconv.ParseUint(v, 0, 64)
	case Float32:
		var tmp float64
		tmp, err = strconv.ParseFloat(v, 64)
		res = float32(tmp)
	default:
		panic(fmt.Sprintf("kind %v is not numeric", k))
	}
	if err != nil {
		panic(fmt.Sprintf("illegal string to numeric %v", k))
	}
	return res

}

func Template[T any](experiment T) []T {
	t := reflect.TypeOf(experiment)
	if !(t.Kind() == reflect.Struct || t.Kind() == reflect.Pointer) {
		panic("template must be a struct (or a pointer to struct)")
	}
	// todo:: is there a bug here? what happens if I pass a pointer to an integer?
	// maybe after the IF below I should recoursively call this function?
	if t.Kind() == reflect.Pointer {
		// this line is to make sure that the ref to experiment passed by the client is not
		// modified here, in case the client relies on it for some reason.
		// 'experiment' is a pointer type so we create New(*experiment)
		newV := reflect.New(reflect.TypeOf(experiment).Elem())
		// but, reflect.New(X) returns *X
		// so we set *newV = *experiment
		newV.Elem().Set(reflect.ValueOf(experiment).Elem())
		experiment = newV.Interface().(T)
		t = t.Elem()
	}

	fieldVals := make([]fieldInfo, 0)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// non-exported fields are left to default value
		// still allowed so coller can use these to store results
		if !f.IsExported() {
			privateVal := maybeDeref(reflect.ValueOf(experiment)).Field(i)
			// val that is same as private val but does not have private origins
			// deepEqual calls value of on its arguments (of type any). so only expects actual types not
			// reflect types. Interface() on a private field panics, so the above copies value of privateVal
			// into a temporary (which then is Interface()able)
			if !privateVal.IsZero() {
				fmt.Println(f.Type, f.Name)
				fmt.Println(privateVal.String() == reflect.Zero(privateVal.Type()).String(), privateVal.Type())
				fmt.Printf("haha %#v %#v\n", privateVal, reflect.Zero(privateVal.Type()))
				panic("private field with non-default value in template not allowed.\n" +
					"Use private fields only to store experiment results")

			}
			continue
		}
		if !(f.Type.ConvertibleTo(reflect.TypeOf(42)) || f.Type == reflect.TypeOf("42") || f.Type == reflect.TypeOf(true)) {
			panic(fmt.Sprintf("only numeric, string and boolean types can be generated in the template. given:%s", f.Type))
		}
		switch f.Type.Kind() {
		case reflect.Bool:
			vals := []interface{}{true, false}
			if v, ok := f.Tag.Lookup(TagName); ok {
				noWhitespace := regexp.MustCompile(`\s`)
				v = noWhitespace.ReplaceAllString(v, "")
				if !(strings.Contains(v, "true") && strings.Contains(v, "false")) {
					vals = []interface{}{strings.Contains(v, "true")}
				}
			}
			fieldVals = append(fieldVals, fieldInfo{f.Name, vals})
		case reflect.String:
			spliter := regexp.MustCompile(`\s*,\s*`)
			noWhitespace := regexp.MustCompile(`^\s|\s$`)
			constraints, ok := f.Tag.Lookup(TagName)
			constraints = noWhitespace.ReplaceAllString(constraints, "")
			valOptions := spliter.Split(constraints, -1)
			if !ok {
				panic(fmt.Sprintf("missing vals tag in field '%s'", f.Name))
			}
			var vals []interface{}
			for _, s := range valOptions {
				vals = append(vals, s)
			}

			fieldVals = append(fieldVals, fieldInfo{f.Name, vals})
		// with the filtering above, these should all be numeric types
		default:
			constraints, ok := f.Tag.Lookup(TagName)
			if !ok {
				panic(fmt.Sprintf("missing vals tag in struct field '%s'\n"+
					"non boolean experiment types must come with a 'vals' tag indicating value range.\n"+
					"e.g. `vals: 1,3,4,5` or `vals:range(40, 100, 25)`", f.Name))
			}

			noWhitespace := regexp.MustCompile(`\s`)
			constraints = noWhitespace.ReplaceAllString(constraints, "")
			if strings.HasPrefix(constraints, "range") {
				panic("found range, doing nothing about it for now")
			} else {
				vals := strings.Split(constraints, ",")
				numVals := []interface{}{}
				for _, v := range vals {
					numVals = append(numVals, parseNumber(v, f.Type.Kind()))
				}
				fieldVals = append(fieldVals, fieldInfo{f.Name, numVals})
			}
		}
	}
	numExps := 1
	for _, fv := range fieldVals {
		numExps *= len(fv.Vals)
	}

	res := fieldPerm(fieldVals, reflect.ValueOf(&experiment).Elem(), []T{}, 0)
	if len(res) != numExps {
		panic(fmt.Sprintf("algorithm issue, expected %d experiments but got %d\n", numExps, len(res)))
	}

	return res
}

func TemplateType[T any]() []T {
	var exp T
	if reflect.TypeOf(exp).Kind() == reflect.Pointer {
		// if T is a pointer type, exp above would be nil
		// to fix, dereference the type, allocate space for it,
		// and assign the updated variable to exp
		// we want a pointer to underlying value of T
		exp = reflect.New(reflect.TypeOf(exp).Elem()).Interface().(T)
	}
	return Template(exp)
}

// dereferences reflect type or value if it is a pointer type
func maybeDeref[TypOrVal interface {
	Kind() reflect.Kind
	Elem() TypOrVal
}](v TypOrVal) TypOrVal {
	if v.Kind() == reflect.Pointer {
		return v.Elem()
	}
	return v
}

func fieldPerm[T any](fieldVals []fieldInfo, exp reflect.Value, acc []T, i int) []T {
	field := fieldVals[i]
	for _, v := range field.Vals {
		if exp.Kind() == reflect.Pointer {
			var tmpExp T
			newV := reflect.New(reflect.TypeOf(tmpExp).Elem())
			newV.Elem().Set(exp.Elem())
			exp = newV
		}
		// . N.B: you might think the above pointer check condition could be in the branch doing
		// appends() and that would be safe as that is the only one actually putting things on the array
		// BUT it is not since previous recoursion layers set up the rest of the struct and the following line
		// will override their results if we do not take care to copy the object first.
		maybeDeref(exp).FieldByName(field.Name).Set(reflect.ValueOf(v))
		if i == len(fieldVals)-1 {
			acc = append(acc, exp.Interface().(T))
		} else {
			acc = fieldPerm(fieldVals, exp, acc, i+1)
		}
	}
	return acc
}

func ToCSV[T any](arr []T) string {
	var tmpExp T
	t := maybeDeref(reflect.TypeOf(tmpExp))
	filename := fmt.Sprintf("%s.csv", t.Name())
	ToCSVPath(filename, arr)

	str := string(Must(io.ReadAll(Must(os.Open(filename)))))
	fmt.Print("*** Experiment results ***\n\n")
	fmt.Println(str)
	return str
}

func ToCSVPath[T any](path string, arr []T) {
	f, err := os.Create(path)
	if err != nil {
		panic(fmt.Sprintf("unable to create file: %v", err))
	}
	defer f.Close()
	ToCSVWriter(f, arr)
}

func ToCSVWriter[T any](w io.Writer, arr []T) {
	csvw := csv.NewWriter(w)
	var tmpExp T
	t := reflect.TypeOf(tmpExp)
	if !(t.Kind() == reflect.Struct || t.Kind() == reflect.Pointer) {
		panic("template must be a struct (not even a pointer to struct)")
	}
	t = maybeDeref(t)
	var heading []string
	for i := 0; i < t.NumField(); i++ {
		heading = append(heading, t.Field(i).Name)
	}
	csvw.Write(heading)
	for _, exp := range arr {
		var row []string
		for i := 0; i < t.NumField(); i++ {
			f := maybeDeref(reflect.ValueOf(exp)).Field(i)
			v := ""
			if f.Kind() == Array || f.Kind() == Slice {
				for i := 0; i < f.Len(); i++ {
					v += fmt.Sprintf("%v", f.Index(i))
					if i != f.Len()-1 {
						v += ","
					}
				}
			} else {
				v = fmt.Sprintf("%v", f)
			}

			row = append(row, v)
		}
		csvw.Write(row)
	}
	csvw.Flush()
}

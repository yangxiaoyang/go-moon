// Moon根据Golang的映射特性，采用了依赖注入的机制。
// inject包提供了多种对实体的映射和依赖注入方式。
package inject

import (
	"fmt"
	"reflect"
)

// Injector接口表示对结构体、函数参数的映射和依赖注入。
type Injector interface {
	Applicator
	Invoker
	TypeMapper
	// SetParent用来设置父injector. 如果在当前injector的Type map中找不到依赖，
	// 将会继续从它的父injector中找，直到返回error.
	SetParent(Injector)
}

// Applicator接口表示到结构体的依赖映射关系。
type Applicator interface {
	// 在Type map中维持对结构体中每个域的引用并用'inject'来标记
	// 如果注入失败将会返回一个error.
	Apply(interface{}) error
}

// Invoker接口表示通过反射进行函数调用。
type Invoker interface {
	// Invoke尝试将interface{}作为一个函数来调用，并基于Type为函数提供参数。
	// 它将返回reflect.Value的切片，其中存放原函数的返回值。
	// 如果注入失败则返回error.
	Invoke(interface{}) ([]reflect.Value, error)
}

// TypeMapper接口用来表示基于类型到接口值的映射。
type TypeMapper interface {
	// 基于调用reflect.TypeOf得到的类型映射interface{}的值。
	Map(interface{}) TypeMapper
	// 基于提供的接口的指针映射interface{}的值。
	// 该函数仅用来将一个值映射为接口，因为接口无法不通过指针而直接引用到。
	MapTo(interface{}, interface{}) TypeMapper
	// 为直接插入基于类型和值的map提供一种可能性。
	// 它使得这一类直接映射成为可能：无法通过反射直接实例化的类型参数，如单向管道。
	Set(reflect.Type, reflect.Value) TypeMapper
	// 返回映射到当前类型的Value. 如果Type没被映射，将返回对应的零值。
	Get(reflect.Type) reflect.Value
}

type injector struct {
	values map[reflect.Type]reflect.Value
	parent Injector
}

// 函数InterfaceOf返回指向接口类型的指针。
// 如果传入的value值不是指向接口的指针，将抛出一个panic异常。
func InterfaceOf(value interface{}) reflect.Type {
	t := reflect.TypeOf(value)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Interface {
		panic("Called inject.InterfaceOf with a value that is not a pointer to an interface. (*MyInterface)(nil)")
	}

	return t
}

// New创建并返回一个Injector.
func New() Injector {
	return &injector{
		values: make(map[reflect.Type]reflect.Value),
	}
}

// Invoke attempts to call the interface{} provided as a function,
// providing dependencies for function arguments based on Type.
// Returns a slice of reflect.Value representing the returned values of the function.
// Returns an error if the injection fails.
// It panics if f is not a function
func (inj *injector) Invoke(f interface{}) ([]reflect.Value, error) {
	t := reflect.TypeOf(f)

	var in = make([]reflect.Value, t.NumIn()) //Panic if t is not kind of Func
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		val := inj.Get(argType)
		if !val.IsValid() {
			return nil, fmt.Errorf("Value not found for type %v", argType)
		}

		in[i] = val
	}

	return reflect.ValueOf(f).Call(in), nil
}

// Maps dependencies in the Type map to each field in the struct
// that is tagged with 'inject'.
// Returns an error if the injection fails.
func (inj *injector) Apply(val interface{}) error {
	v := reflect.ValueOf(val)

	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil // Should not panic here ?
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		structField := t.Field(i)
		if f.CanSet() && (structField.Tag == "inject" || structField.Tag.Get("inject") != "") {
			ft := f.Type()
			v := inj.Get(ft)
			if !v.IsValid() {
				return fmt.Errorf("Value not found for type %v", ft)
			}

			f.Set(v)
		}

	}

	return nil
}

// Maps the concrete value of val to its dynamic type using reflect.TypeOf,
// It returns the TypeMapper registered in.
func (i *injector) Map(val interface{}) TypeMapper {
	i.values[reflect.TypeOf(val)] = reflect.ValueOf(val)
	return i
}

func (i *injector) MapTo(val interface{}, ifacePtr interface{}) TypeMapper {
	i.values[InterfaceOf(ifacePtr)] = reflect.ValueOf(val)
	return i
}

// Maps the given reflect.Type to the given reflect.Value and returns
// the Typemapper the mapping has been registered in.
func (i *injector) Set(typ reflect.Type, val reflect.Value) TypeMapper {
	i.values[typ] = val
	return i
}

func (i *injector) Get(t reflect.Type) reflect.Value {
	val := i.values[t]

	if val.IsValid() {
		return val
	}

	// no concrete types found, try to find implementors
	// if t is an interface
	if t.Kind() == reflect.Interface {
		for k, v := range i.values {
			if k.Implements(t) {
				val = v
				break
			}
		}
	}

	// Still no type found, try to look it up on the parent
	if !val.IsValid() && i.parent != nil {
		val = i.parent.Get(t)
	}

	return val

}

// SetParent用来设置父injector. 如果在当前injector的Type map中找不到依赖，
// 将会继续从它的父injector中找，直到返回error.
func (i *injector) SetParent(parent Injector) {
	i.parent = parent
}

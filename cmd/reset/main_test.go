package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	t.Parallel()

	t.Run("возвращает поля структуры", func(t *testing.T) {
		t.Parallel()

		spec := &ast.TypeSpec{
			Type: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{
								{Name: "name"},
							},
							Type: &ast.Ident{Name: "string"},
						},
						{
							Names: []*ast.Ident{
								{Name: "age"},
							},
							Type: &ast.Ident{Name: "int"},
						},
					},
				},
			},
		}

		result := fields(spec)

		require.Len(t, result, 2)
		assert.Equal(t, "name", result[0].Names[0].Name)
		assert.Equal(t, "age", result[1].Names[0].Name)
	})

	t.Run("возвращает nil для не структуры", func(t *testing.T) {
		t.Parallel()

		spec := &ast.TypeSpec{
			Type: &ast.Ident{Name: "string"},
		}

		result := fields(spec)

		assert.Nil(t, result)
	})

	t.Run("возвращает пустой список для структуры без полей", func(t *testing.T) {
		t.Parallel()

		spec := &ast.TypeSpec{
			Type: &ast.StructType{
				Fields: &ast.FieldList{},
			},
		}

		result := fields(spec)

		assert.Empty(t, result)
	})
}

func TestFieldNames(t *testing.T) {
	t.Parallel()

	t.Run("возвращает имя одного поля", func(t *testing.T) {
		t.Parallel()

		field := &ast.Field{
			Names: []*ast.Ident{
				{Name: "age"},
			},
			Type: &ast.Ident{Name: "int"},
		}

		result := fieldNames(field)

		assert.Equal(t, []string{"age"}, result)
	})

	t.Run("возвращает имена нескольких полей", func(t *testing.T) {
		t.Parallel()

		field := &ast.Field{
			Names: []*ast.Ident{
				{Name: "name"},
				{Name: "surname"},
			},
			Type: &ast.Ident{Name: "string"},
		}

		result := fieldNames(field)

		assert.Equal(t, []string{"name", "surname"}, result)
	})

	t.Run("возвращает пустой список для поля без имени", func(t *testing.T) {
		t.Parallel()

		field := &ast.Field{
			Type: &ast.Ident{Name: "string"},
		}

		result := fieldNames(field)

		require.NotNil(t, result)
		assert.Empty(t, result)
	})
}

func TestResetValue(t *testing.T) {
	t.Parallel()

	t.Run("сбрасывает строку", func(t *testing.T) {
		t.Parallel()

		result := resetValue("m.name", &ast.Ident{Name: "string"})

		assert.Equal(t, `m.name = ""`, result)
	})

	t.Run("сбрасывает bool", func(t *testing.T) {
		t.Parallel()

		result := resetValue("m.enabled", &ast.Ident{Name: "bool"})

		assert.Equal(t, "m.enabled = false", result)
	})

	t.Run("сбрасывает числовые типы", func(t *testing.T) {
		t.Parallel()

		types := []string{
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"uintptr", "float32", "float64",
		}

		for _, typeName := range types {
			t.Run(typeName, func(t *testing.T) {
				t.Parallel()

				result := resetValue("m.value", &ast.Ident{Name: typeName})

				assert.Equal(t, "m.value = 0", result)
			})
		}
	})

	t.Run("обнуляет длину среза", func(t *testing.T) {
		t.Parallel()

		expr := &ast.ArrayType{
			Elt: &ast.Ident{Name: "int"},
		}

		result := resetValue("m.items", expr)

		assert.Equal(t, "m.items = m.items[:0]", result)
	})

	t.Run("очищает мапу", func(t *testing.T) {
		t.Parallel()

		expr := &ast.MapType{
			Key:   &ast.Ident{Name: "string"},
			Value: &ast.Ident{Name: "int"},
		}

		result := resetValue("m.values", expr)

		assert.Equal(t, "clear(m.values)", result)
	})

	t.Run("не обрабатывает массив", func(t *testing.T) {
		t.Parallel()

		expr := &ast.ArrayType{
			Len: &ast.BasicLit{
				Kind:  token.INT,
				Value: "3",
			},
			Elt: &ast.Ident{Name: "int"},
		}

		result := resetValue("m.items", expr)

		assert.Empty(t, result)
	})

	t.Run("не обрабатывает неизвестный тип", func(t *testing.T) {
		t.Parallel()

		result := resetValue("m.value", &ast.Ident{Name: "CustomType"})

		assert.Empty(t, result)
	})

	t.Run("не обрабатывает структуру", func(t *testing.T) {
		t.Parallel()

		result := resetValue("m.child", &ast.StructType{
			Fields: &ast.FieldList{},
		})

		assert.Empty(t, result)
	})
}

func TestHasResetMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		typeName string
		expected bool
	}{
		{
			name: "структура с методом Reset",
			source: `
package test

type Example struct{}

func (e *Example) Reset() {}
`,
			typeName: "Example",
			expected: true,
		},
		{
			name: "структура с Reset на значении",
			source: `
package test

type Example struct{}

func (e Example) Reset() {}
`,
			typeName: "Example",
			expected: true,
		},
		{
			name: "структура без метода Reset",
			source: `
package test

type Example struct{}

func (e *Example) Clear() {}
`,
			typeName: "Example",
			expected: false,
		},
		{
			name: "Reset с аргументами",
			source: `
package test

type Example struct{}

func (e *Example) Reset(value int) {}
`,
			typeName: "Example",
			expected: false,
		},
		{
			name: "Reset с возвращаемым значением",
			source: `
package test

type Example struct{}

func (e *Example) Reset() error {
	return nil
}
`,
			typeName: "Example",
			expected: false,
		},
		{
			name: "не структура",
			source: `
package test

type Example int

func (e *Example) Reset() {}
`,
			typeName: "Example",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()

			file, err := parser.ParseFile(
				fset,
				"test.go",
				tt.source,
				0,
			)
			require.NoError(t, err)

			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
			}

			_, err = (&types.Config{}).Check(
				"test",
				fset,
				[]*ast.File{file},
				info,
			)
			require.NoError(t, err)

			var expr ast.Expr

			ast.Inspect(file, func(node ast.Node) bool {
				spec, ok := node.(*ast.TypeSpec)
				if !ok || spec.Name.Name != tt.typeName {
					return true
				}

				obj := info.Defs[spec.Name]
				require.NotNil(t, obj)

				expr = &ast.Ident{Name: spec.Name.Name}
				info.Types[expr] = types.TypeAndValue{
					Type: obj.Type(),
				}

				return false
			})

			require.NotNil(t, expr)

			t.Logf("expr: %T", expr)
			t.Logf("type: %T, value: %v", info.TypeOf(expr), info.TypeOf(expr))
			result := hasResetMethod(expr, info)

			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("возвращает false для неизвестного типа", func(t *testing.T) {
		t.Parallel()

		info := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}

		result := hasResetMethod(
			&ast.Ident{Name: "Unknown"},
			info,
		)

		assert.False(t, result)
	})
}

func TestResetField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		field    string
		expected string
	}{
		{
			name: "сбрасывает int",
			source: `
package test
type Example struct {
	age int
}`,
			field:    "age",
			expected: "m.age = 0",
		},
		{
			name: "сбрасывает string",
			source: `
package test
type Example struct {
	name string
}`,
			field:    "name",
			expected: `m.name = ""`,
		},
		{
			name: "сбрасывает bool",
			source: `
package test
type Example struct {
	enabled bool
}`,
			field:    "enabled",
			expected: "m.enabled = false",
		},
		{
			name: "обнуляет длину среза",
			source: `
package test
type Example struct {
	items []int
}`,
			field:    "items",
			expected: "m.items = m.items[:0]",
		},
		{
			name: "очищает мапу",
			source: `
package test
type Example struct {
	values map[string]int
}`,
			field:    "values",
			expected: "clear(m.values)",
		},
		{
			name: "сбрасывает указатель на int",
			source: `
package test
type Example struct {
	age *int
}`,
			field: "age",
			expected: `if m.age != nil {
(*m.age) = 0
}`,
		},
		{
			name: "сбрасывает указатель на string",
			source: `
package test
type Example struct {
	name *string
}`,
			field: "name",
			expected: `if m.name != nil {
(*m.name) = ""
}`,
		},
		{
			name: "сбрасывает указатель на срез",
			source: `
package test
type Example struct {
	items *[]int
}`,
			field: "items",
			expected: `if m.items != nil {
(*m.items) = (*m.items)[:0]
}`,
		},
		{
			name: "очищает мапу по указателю",
			source: `
package test
type Example struct {
	values *map[string]int
}`,
			field: "values",
			expected: `if m.values != nil {
clear((*m.values))
}`,
		},
		{
			name: "вызывает Reset у вложенной структуры",
			source: `
package test

type Child struct{}

func (c *Child) Reset() {}

type Example struct {
	child Child
}`,
			field:    "child",
			expected: "m.child.Reset()",
		},
		{
			name: "вызывает Reset у указателя на структуру",
			source: `
package test

type Child struct{}

func (c *Child) Reset() {}

type Example struct {
	child *Child
}`,
			field: "child",
			expected: `if m.child != nil {
	m.child.Reset()
}`,
		},
		{
			name: "проверяет Reset у интерфейса",
			source: `
package test

type Example struct {
	child interface{}
}`,
			field: "child",
			expected: `if resetter, ok := m.child.(interface{ Reset() }); ok && resetter != nil {
	resetter.Reset()
}`,
		},
		{
			name: "пропускает структуру без Reset",
			source: `
package test

type Child struct{}

type Example struct {
	child Child
}`,
			field:    "child",
			expected: "",
		},
		{
			name: "пропускает массив",
			source: `
package test

type Example struct {
	items [3]int
}`,
			field:    "items",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()

			file, err := parser.ParseFile(
				fset,
				"test.go",
				tt.source,
				0,
			)
			require.NoError(t, err)

			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
			}

			_, err = (&types.Config{}).Check(
				"test",
				fset,
				[]*ast.File{file},
				info,
			)
			require.NoError(t, err)

			var fieldType ast.Expr

			ast.Inspect(file, func(node ast.Node) bool {
				spec, ok := node.(*ast.TypeSpec)
				if !ok || spec.Name.Name != "Example" {
					return true
				}

				structType, ok := spec.Type.(*ast.StructType)
				if !ok {
					return false
				}

				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						if name.Name == tt.field {
							fieldType = field.Type
							return false
						}
					}
				}

				return false
			})

			require.NotNil(t, fieldType)

			result := resetField(tt.field, fieldType, info)

			assert.Equal(
				t,
				strings.TrimSpace(tt.expected),
				strings.TrimSpace(result),
			)
		})
	}

	t.Run("неизвестный тип", func(t *testing.T) {
		t.Parallel()

		info := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}

		result := resetField(
			"value",
			&ast.Ident{Name: "Unknown"},
			info,
		)

		assert.Empty(t, result)
	})
}

func TestGenerateReset(t *testing.T) {
	t.Parallel()

	t.Run("генерирует Reset для структуры с полями", func(t *testing.T) {
		t.Parallel()

		source := `
package test

type Example struct {
	age     int
	name    string
	enabled bool
	items   []int
	values  map[string]int
}
`

		fset := token.NewFileSet()

		file, err := parser.ParseFile(fset, "test.go", source, 0)
		require.NoError(t, err)

		info := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}

		_, err = (&types.Config{}).Check(
			"test",
			fset,
			[]*ast.File{file},
			info,
		)
		require.NoError(t, err)

		var spec *ast.TypeSpec

		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, ok := node.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Example" {
				return true
			}

			spec = typeSpec
			return false
		})

		require.NotNil(t, spec)

		var buf bytes.Buffer

		err = generateReset(spec, info, &buf)
		require.NoError(t, err)

		expected := `
func (m *Example) Reset() {
	if m == nil {
		return
	}

	m.age = 0
	m.name = ""
	m.enabled = false
	m.items = m.items[:0]
	clear(m.values)
}
`

		actual, err := format.Source(buf.Bytes())
		require.NoError(t, err)

		formattedExpected, err := format.Source([]byte(expected))
		require.NoError(t, err)

		assert.Equal(t, string(formattedExpected), string(actual))
	})

	t.Run("генерирует Reset для пустой структуры", func(t *testing.T) {
		t.Parallel()

		spec := &ast.TypeSpec{
			Name: &ast.Ident{Name: "Empty"},
			Type: &ast.StructType{
				Fields: &ast.FieldList{},
			},
		}

		var buf bytes.Buffer

		err := generateReset(spec, &types.Info{}, &buf)
		require.NoError(t, err)

		expected := `
func (m *Empty) Reset() {
	if m == nil {
		return
	}
}
`

		actual, err := format.Source(buf.Bytes())
		require.NoError(t, err)

		formattedExpected, err := format.Source([]byte(expected))
		require.NoError(t, err)

		assert.Equal(t, string(formattedExpected), string(actual))
	})

	t.Run("генерирует несколько методов в один буфер", func(t *testing.T) {
		t.Parallel()

		info := &types.Info{}

		specs := []*ast.TypeSpec{
			{
				Name: &ast.Ident{Name: "First"},
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
			{
				Name: &ast.Ident{Name: "Second"},
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
		}

		var buf bytes.Buffer

		for _, spec := range specs {
			err := generateReset(spec, info, &buf)
			require.NoError(t, err)
		}

		actual, err := format.Source(buf.Bytes())
		require.NoError(t, err)

		assert.Contains(t, string(actual), "func (m *First) Reset()")
		assert.Contains(t, string(actual), "func (m *Second) Reset()")
	})
}

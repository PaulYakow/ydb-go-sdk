package params

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"

	"github.com/ydb-platform/ydb-go-sdk/v3/internal/allocator"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/value"
	"github.com/ydb-platform/ydb-go-sdk/v3/internal/xtest"
)

func TestDict(t *testing.T) {
	type expected struct {
		kind  *Ydb.Type
		value *Ydb.Value
	}

	tests := []struct {
		method string
		args   []any

		expected expected
	}{
		{
			method: "Uint64",
			args:   []any{uint64(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_UINT64},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint64Value{
						Uint64Value: 123,
					},
				},
			},
		},
		{
			method: "Int64",
			args:   []any{int64(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_INT64},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Int64Value{
						Int64Value: 123,
					},
				},
			},
		},
		{
			method: "Uint32",
			args:   []any{uint32(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_UINT32},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint32Value{
						Uint32Value: 123,
					},
				},
			},
		},
		{
			method: "Int32",
			args:   []any{int32(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_INT32},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Int32Value{
						Int32Value: 123,
					},
				},
			},
		},
		{
			method: "Uint16",
			args:   []any{uint16(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_UINT16},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint32Value{
						Uint32Value: 123,
					},
				},
			},
		},
		{
			method: "Int16",
			args:   []any{int16(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_INT16},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Int32Value{
						Int32Value: 123,
					},
				},
			},
		},
		{
			method: "Uint8",
			args:   []any{uint8(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_UINT8},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint32Value{
						Uint32Value: 123,
					},
				},
			},
		},
		{
			method: "Int8",
			args:   []any{int8(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_INT8},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Int32Value{
						Int32Value: 123,
					},
				},
			},
		},
		{
			method: "Bool",
			args:   []any{true},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_BOOL},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_BoolValue{
						BoolValue: true,
					},
				},
			},
		},
		{
			method: "Text",
			args:   []any{"test"},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_UTF8},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_TextValue{
						TextValue: "test",
					},
				},
			},
		},
		{
			method: "Bytes",
			args:   []any{[]byte("test")},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_STRING},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_BytesValue{
						BytesValue: []byte("test"),
					},
				},
			},
		},
		{
			method: "Float",
			args:   []any{float32(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_FLOAT},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_FloatValue{
						FloatValue: float32(123),
					},
				},
			},
		},
		{
			method: "Double",
			args:   []any{float64(123)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_DOUBLE},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_DoubleValue{
						DoubleValue: float64(123),
					},
				},
			},
		},
		{
			method: "Interval",
			args:   []any{time.Second},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_INTERVAL},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Int64Value{
						Int64Value: 1000000,
					},
				},
			},
		},
		{
			method: "Datetime",
			args:   []any{time.Unix(123456789, 456)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_DATETIME},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint32Value{
						Uint32Value: 123456789,
					},
				},
			},
		},
		{
			method: "Date",
			args:   []any{time.Unix(123456789, 456)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_DATE},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint32Value{
						Uint32Value: 1428,
					},
				},
			},
		},
		{
			method: "Timestamp",
			args:   []any{time.Unix(123456789, 456)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_TIMESTAMP},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Uint64Value{
						Uint64Value: 123456789000000,
					},
				},
			},
		},
		{
			method: "Decimal",
			args:   []any{[...]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6}, uint32(22), uint32(9)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_DecimalType{
						DecimalType: &Ydb.DecimalType{
							Precision: 22,
							Scale:     9,
						},
					},
				},
				value: &Ydb.Value{
					High_128: 72623859790382856,
					Value: &Ydb.Value_Low_128{
						Low_128: 648519454493508870,
					},
				},
			},
		},
		{
			method: "JSON",
			args:   []any{`{"a": 1,"b": "B"}`},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_JSON},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_TextValue{
						TextValue: `{"a": 1,"b": "B"}`,
					},
				},
			},
		},
		{
			method: "JSONDocument",
			args:   []any{`{"a": 1,"b": "B"}`},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_JSON_DOCUMENT},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_TextValue{
						TextValue: `{"a": 1,"b": "B"}`,
					},
				},
			},
		},
		{
			method: "YSON",
			args:   []any{[]byte(`{"a": 1,"b": "B"}`)},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_YSON},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_BytesValue{
						BytesValue: []byte(`{"a": 1,"b": "B"}`),
					},
				},
			},
		},
		{
			method: "UUID",
			args:   []any{[...]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},

			expected: expected{
				kind: &Ydb.Type{
					Type: &Ydb.Type_TypeId{TypeId: Ydb.Type_UUID},
				},
				value: &Ydb.Value{
					Value: &Ydb.Value_Low_128{
						Low_128: 651345242494996240,
					},
					High_128: 72623859790382856,
				},
			},
		},
	}

	for _, key := range tests {
		for _, val := range tests {
			t.Run(fmt.Sprintf("%s:%s", key.method, val.method), func(t *testing.T) {
				a := allocator.New()
				defer a.Free()

				item := Builder{}.Param("$x").BeginDict().Add()

				addedKey, ok := xtest.CallMethod(item, key.method, key.args...)[0].(*dictValue)
				require.True(t, ok)

				d, ok := xtest.CallMethod(addedKey, val.method, val.args...)[0].(*dict)
				require.True(t, ok)

				params := d.EndDict().Build().ToYDB(a)
				require.Equal(t, paramsToJSON(
					map[string]*Ydb.TypedValue{
						"$x": {
							Type: &Ydb.Type{
								Type: &Ydb.Type_DictType{
									DictType: &Ydb.DictType{
										Key:     key.expected.kind,
										Payload: val.expected.kind,
									},
								},
							},
							Value: &Ydb.Value{
								Pairs: []*Ydb.ValuePair{
									{
										Key:     key.expected.value,
										Payload: val.expected.value,
									},
								},
							},
						},
					}), paramsToJSON(params))
			})
		}
	}
}

func TestDict_AddPairs(t *testing.T) {
	a := allocator.New()
	defer a.Free()

	pairs := []value.DictValueField{
		{
			K: value.Int64Value(123),
			V: value.BoolValue(true),
		},
		{
			K: value.Int64Value(321),
			V: value.BoolValue(false),
		},
	}

	params := Builder{}.Param("$x").BeginDict().AddPairs(pairs...).EndDict().Build().ToYDB(a)

	require.Equal(t, paramsToJSON(
		map[string]*Ydb.TypedValue{
			"$x": {
				Type: &Ydb.Type{
					Type: &Ydb.Type_DictType{
						DictType: &Ydb.DictType{
							Key: &Ydb.Type{
								Type: &Ydb.Type_TypeId{
									TypeId: Ydb.Type_INT64,
								},
							},
							Payload: &Ydb.Type{
								Type: &Ydb.Type_TypeId{
									TypeId: Ydb.Type_BOOL,
								},
							},
						},
					},
				},
				Value: &Ydb.Value{
					Pairs: []*Ydb.ValuePair{
						{
							Key: &Ydb.Value{
								Value: &Ydb.Value_Int64Value{
									Int64Value: 123,
								},
							},
							Payload: &Ydb.Value{
								Value: &Ydb.Value_BoolValue{
									BoolValue: true,
								},
							},
						},
						{
							Key: &Ydb.Value{
								Value: &Ydb.Value_Int64Value{
									Int64Value: 321,
								},
							},
							Payload: &Ydb.Value{
								Value: &Ydb.Value_BoolValue{
									BoolValue: false,
								},
							},
						},
					},
				},
			},
		}), paramsToJSON(params))
}

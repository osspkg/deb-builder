package utils

import (
	"testing"
)

func TestUnit_MustValueAfterPrefix(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		prefix string
		want   string
		panic  bool // ожидается ли паника
	}{
		// успешные случаи
		{"префикс в начале", "hello world", "hello", " world", false},
		{"префикс в середине", "xxhello world", "hello", " world", false},
		{"пустой префикс", "value", "", "value", false},
		{"оба параметра пусты", "", "", "", false},
		{"возврат пустой строки (префикс равен значению)", "whole", "whole", "", false},
		{"юникод", "привет мир", "привет", " мир", false},
		{"несколько вхождений – берётся первое", "a_b_c", "_", "b_c", false},
		{"префикс, который встречается несколько раз", "prefixprefixsuffix", "prefix", "prefixsuffix", false},

		// случаи с паникой
		{"префикс не найден", "value", "missing", "", true},
		{"пустое значение и непустой префикс", "", "x", "", true},
		{"префикс длиннее значения", "abc", "abcdef", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("ожидалась паника, но функция завершилась без паники")
					}
				}()
				_ = MustValueAfterPrefix(tt.value, tt.prefix)
				t.Error("ожидалась паника, но функция вернула управление")
				return
			}

			got := MustValueAfterPrefix(tt.value, tt.prefix)
			if got != tt.want {
				t.Errorf("MustValueAfterPrefix(%q, %q) = %q, want %q", tt.value, tt.prefix, got, tt.want)
			}
		})
	}
}

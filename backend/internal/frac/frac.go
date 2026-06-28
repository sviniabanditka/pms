// Package frac — фракционная индексация позиций (строки вместо float).
// Между любыми двумя ключами всегда есть третий → нет исчерпания точности (D20).
// Алфавит base36 ('0'-'9','a'-'z', ASCII-возрастающий). Инвариант результата: нет хвостового '0'.
package frac

import "strings"

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

const base = len(alphabet)

func rank(c byte) int { return strings.IndexByte(alphabet, c) }

// Between возвращает ключ строго между lo и hi (lexicographically lo < результат < hi).
// lo == "" → нижняя граница (минус-бесконечность), hi == "" → верхняя (плюс-бесконечность).
// Предусловие: при обоих непустых lo < hi.
func Between(lo, hi string) string {
	if lo == "" && hi == "" {
		return string(alphabet[base/2]) // первый ключ ~"i"
	}
	var sb []byte
	i := 0
	for {
		l := 0
		if i < len(lo) {
			l = rank(lo[i])
		}
		h := base
		if hi != "" {
			if i < len(hi) {
				h = rank(hi[i])
			} else {
				h = 0 // hi короче — недостижимо при lo<hi, но безопасно
			}
		}
		if l == h {
			sb = append(sb, alphabet[l])
			i++
			continue
		}
		mid := (l + h) / 2
		if mid > l {
			sb = append(sb, alphabet[mid])
			return string(sb)
		}
		// h == l+1: берём нижний разряд и спускаемся (hi-ограничение снято).
		sb = append(sb, alphabet[l])
		i++
		hi = ""
	}
}

// First — ключ для первого элемента пустого списка.
func First() string { return Between("", "") }

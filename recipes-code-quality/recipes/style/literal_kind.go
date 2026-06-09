/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import "strings"

// isStringLiteralSource reports whether the raw literal source is a Go string
// literal — interpreted ("...") or raw (`...`).
func isStringLiteralSource(source string) bool {
	return len(source) > 0 && (source[0] == '"' || source[0] == '`')
}

// isIntLiteralSource reports whether the raw literal source is a Go integer
// literal (decimal, hex, octal or binary), excluding floats, imaginary and
// character/string literals.
func isIntLiteralSource(source string) bool {
	if len(source) == 0 || source[0] < '0' || source[0] > '9' {
		return false
	}
	lower := strings.ToLower(source)
	if strings.ContainsAny(lower, ".i") { // float (1.5) or imaginary (3i)
		return false
	}
	if strings.HasPrefix(lower, "0x") {
		return !strings.Contains(lower, "p") // 'p' marks a hex-float exponent
	}
	return !strings.Contains(lower, "e") // 'e' marks a decimal-float exponent
}

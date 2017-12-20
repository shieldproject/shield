package s3

func uriencode(s string, encodeSlash bool) []byte {
	bb := []byte(s)

	n := 0
	for _, b := range bb {
		/* percent-encode everything that isn't

		   A-Z: 0x41 - 0x5a
		   a-z: 0x61 - 0x7a
		   0-9: 0x30 - 0x39
		   -:   0x2d
		   .:   0x2e
		   _:   0x5f
		   ~:   0x7e
		   /:   0x2f (but only if encodeSlash == true)

		*/

		switch {
		case b >= 0x41 && b <= 0x5a:
			n++
		case b >= 0x61 && b <= 0x7a:
			n++
		case b >= 0x30 && b <= 0x39:
			n++
		case b == 0x2d || b == 0x2e || b == 0x5f || b == 0x7e:
			n++
		case b == 0x2f && !encodeSlash:
			n++
		default: /* %xx-encoded */
			n += 3
		}
	}

	out := make([]byte, n)
	n = 0

	for _, b := range bb {
		switch {
		case b >= 0x41 && b <= 0x5a:
			out[n] = b
			n++
		case b >= 0x61 && b <= 0x7a:
			out[n] = b
			n++
		case b >= 0x30 && b <= 0x39:
			out[n] = b
			n++
		case b == 0x2d || b == 0x2e || b == 0x5f || b == 0x7e:
			out[n] = b
			n++
		case b == 0x2f && !encodeSlash:
			out[n] = b
			n++
		default: /* %xx-encoded */
			out[n] = 0x25 /* '%' */
			out[n+1] = "0123456789ABCDEF"[b>>4]
			out[n+2] = "0123456789ABCDEF"[b&0xf]
			n += 3
		}
	}

	return out
}

func hex(b []byte) string {
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = "0123456789abcdef"[c>>4]
		out[i*2+1] = "0123456789abcdef"[c&0xf]
	}
	return string(out)
}

func base64(b []byte) []byte {
	/* per RFC2054 */
	alpha := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	out := make([]byte, ((len(b) + 2) / 3 * 4))

	i, j := 0, 0
	n := len(b) / 3 * 3

	for i < n {
		v := uint(b[i])<<16 | uint(b[i+1])<<8 | uint(b[i+2])
		out[j+0] = alpha[v>>18&0x3f]
		out[j+1] = alpha[v>>12&0x3f]
		out[j+2] = alpha[v>>6&0x3f]
		out[j+3] = alpha[v&0x3f]
		i += 3
		j += 4
	}

	left := len(b) - n
	if left > 0 {
		v := uint(b[i]) << 16
		if left == 2 {
			v |= uint(b[i+1]) << 8
		}
		out[j+0] = alpha[v>>18&0x3f]
		out[j+1] = alpha[v>>12&0x3f]
		if left == 2 {
			out[j+2] = alpha[v>>6&0x3f]
		} else {
			out[j+2] = alpha[64]
		}
		out[j+3] = alpha[64]
	}
	return out
}

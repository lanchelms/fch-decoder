package valheim

import "github.com/lanchelms/fch-decoder/binary"

type decoder interface {
	Decode(*binary.Reader)
}

type encoder interface {
	Encode(*binary.Writer)
}

type pointerDecoder[T any] interface {
	*T
	decoder
}

func readValue[T any, P pointerDecoder[T]](r *binary.Reader) T {
	var value T
	P(&value).Decode(r)
	return value
}

func readList[T any, P pointerDecoder[T]](r *binary.Reader) []T {
	count := r.Uint32()
	out := make([]T, 0, r.Capacity(count))
	for range count {
		out = append(out, readValue[T, P](r))
	}
	return out
}

func writeList[T encoder](w *binary.Writer, values []T) {
	w.Uint32(uint32(len(values)))
	for _, value := range values {
		value.Encode(w)
	}
}

func readStringList(r *binary.Reader) []string {
	count := r.Uint32()
	out := make([]string, 0, r.Capacity(count))
	for range count {
		out = append(out, r.String())
	}
	return out
}

func writeStringList(w *binary.Writer, values []string) {
	w.Uint32(uint32(len(values)))
	for _, value := range values {
		w.String(value)
	}
}

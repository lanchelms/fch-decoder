package fch

type pointerDecoder[T any] interface {
	*T
	decoder
}

func readValue[T any, P pointerDecoder[T]](r *Reader) T {
	var value T
	P(&value).Decode(r)
	return value
}

func readList[T any, P pointerDecoder[T]](r *Reader) []T {
	count := r.u32()
	out := make([]T, 0, r.capacity(count))
	for range count {
		out = append(out, readValue[T, P](r))
	}
	return out
}

func writeList[T encoder](w *Writer, values []T) {
	w.u32(uint32(len(values)))
	for _, value := range values {
		value.Encode(w)
	}
}

func readStringList(r *Reader) []string {
	count := r.u32()
	out := make([]string, 0, r.capacity(count))
	for range count {
		out = append(out, r.str())
	}
	return out
}

func writeStringList(w *Writer, values []string) {
	w.u32(uint32(len(values)))
	for _, value := range values {
		w.str(value)
	}
}

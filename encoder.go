package fch

import "io"

type encoder interface {
	Encode(*Writer)
}

func Encode(w io.Writer, c *Character) error {
	data, err := EncodeBytes(c)
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func EncodeBytes(c *Character) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	w := NewWriter()
	c.Encode(w)
	return w.Data(), nil
}

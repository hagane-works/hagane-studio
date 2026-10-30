package exportengine

import "fmt"

func ResolveImageDDSTolerant(resolve BlobResolver, key, label string) ([]byte, bool, error) {
	if resolve == nil {

		return nil, false, fmt.Errorf("%s requires private object resolver", label)
	}
	dds, err := resolveImageDDS(resolve, key, label)
	if err != nil {

		return nil, false, nil
	}
	return dds, true, nil
}

func ResolveBlobTolerant(resolve BlobResolver, key string, maxBytes int64, label string) ([]byte, bool, error) {
	if resolve == nil {
		return nil, false, fmt.Errorf("%s requires private object resolver", label)
	}
	data, err := resolve(key, maxBytes)
	if err != nil {
		return nil, false, nil
	}
	return data, true, nil
}

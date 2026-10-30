package content

func ValidateWargoalSafeScalars(value map[string]string) (map[string]string, error) {
	return wargoalValidateSafeScalars(value)
}

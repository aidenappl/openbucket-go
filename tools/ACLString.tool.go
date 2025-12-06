package tools

func ACLString(isPublic bool) string {
	if isPublic {
		return "public-read"
	}
	return "private"
}

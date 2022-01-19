package endpoint

func FormGalactusURL(address, baseRoute, endpoint string, childPaths ...string) (url string) {
	url = address + baseRoute + endpoint
	for i, v := range childPaths {
		url += v
		if i < len(childPaths)-1 {
			url += "/"
		}
	}
	return
}

package endpoint

func FormGalactusURL(address, baseRoute, endpoint string, childPaths ...string) (url string) {
	url = address + baseRoute + endpoint
	for _, v := range childPaths {
		url += v + "/"
	}
	return
}

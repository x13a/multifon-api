package multifonapi

import "net/url"

func urlJoin(base, urlPath string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	pathURL, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(pathURL).String(), nil
}

package utils

func SetFeature(feature string, value bool) {
	features[feature] = value
}

func GetFeature(feature string) bool {
	return features[feature]
}

func GetFeatureString() string {
	var featureString string
	var i = len(features)
	for feature, value := range features {
		if value {
			featureString += feature
			if i > 1 {
				featureString += ", "
			}
		}
		i--
	}
	return featureString
}

var features = map[string]bool{}

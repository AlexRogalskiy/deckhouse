package utils

import (
	"reflect"
)

type mergeValuesPair struct {
	A map[interface{}]interface{}
	B map[interface{}]interface{}
}

func DeepMerge(maps ...map[interface{}]interface{}) map[interface{}]interface{} {
	res := make(map[interface{}]interface{})
	for _, m := range maps {
		res = deepMergeTwo(res, m)
	}
	return res
}

func deepMergeTwo(A map[interface{}]interface{}, B map[interface{}]interface{}) map[interface{}]interface{} {
	res := make(map[interface{}]interface{})
	for key, value := range A {
		res[key] = value
	}

	queue := []mergeValuesPair{{A: res, B: B}}

	for len(queue) > 0 {
		pair := queue[0]
		queue = queue[1:]

		for k, v2 := range pair.B {
			v1, isExist := pair.A[k]

			if isExist {
				v1Type := reflect.TypeOf(v1)
				v2Type := reflect.TypeOf(v2)

				if (v1Type == v2Type) && (v1Type != nil) {
					switch v1Type.Kind() {
					case reflect.Map:
						resMap := make(map[interface{}]interface{})
						for key, value := range v1.(map[interface{}]interface{}) {
							resMap[key] = value
						}
						pair.A[k] = resMap

						queue = append(queue, mergeValuesPair{
							A: resMap,
							B: v2.(map[interface{}]interface{}),
						})
					case reflect.Array, reflect.Slice:
						resArr := make([]interface{}, 0)
						for _, elem := range v1.([]interface{}) {
							resArr = append(resArr, elem)
						}
						for _, elem := range v2.([]interface{}) {
							resArr = append(resArr, elem)
						}
						pair.A[k] = resArr
					default:
						pair.A[k] = v2
					}
				} else {
					pair.A[k] = v2
				}
			} else {
				pair.A[k] = v2
			}
		}
	}

	return res
}

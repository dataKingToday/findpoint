package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	geojson "github.com/paulmach/go.geojson"
)

//위치(경도,위도) 구조체
type Position struct {
	Longitude float64 `json:"lng"` //경도
	Latitude  float64 `json:"lat"` //위도
}

//geojson의 tag(ex:Link ID) 구조체
type XYZData struct {
	Tags []string `json:"tags"`
}

//FindPoint 수행 결과 구조체
type ResultOfFindPoint struct {
	linkId          string
	linkElement     [][]float64 //link의 point들
	nearestDistance float64     //PH 최단거리
	nearestPosition Position    //H 좌표
	comeOrGoAway    string      //Link의 방향(다가오는지, 멀어지는지)
	isInLink        string      //Link내에 H좌표 유무 여부
}

//Link내 Small Link의 FindPoint 결과 구조체
type ResultOfSmallLink struct {
	distance    float64  //samll link와 좌표P 간의 최단거리
	destination Position //samll link와 좌표P 간의 최단거리를 가지는 지점의 좌표
	isInLink    string   //samll link와 좌표P 간의 직교 지점의 좌표H가 small link내에 존재하는지 여부
}

//Input Link 정보(geojson→GO변환후) 구조체
type InputLink struct {
	linkId           string      //link의 ID
	linkElement      [][]float64 //link의 point의 좌표(경도,위도)
	howManySmallLink int         //Small Link의 개수
}

//geojson→GO 변환 함수
func NewGeoJSON(position Position, tags []string) ([]byte, error) {
	featureCollection := geojson.NewFeatureCollection()
	feature := geojson.NewPointFeature([]float64{position.Longitude, position.Latitude})
	feature.SetProperty("@ns:com:here:xyz", XYZData{Tags: tags})
	featureCollection.AddFeature(feature)
	return featureCollection.MarshalJSON()
}

//개별 Link 데이터 저장함수 (from geojson→GO 변환된 데이터)
func (i InputLink) GetLink(linkId string, linkElement [][]float64) InputLink {
	i.linkId = linkId
	i.linkElement = linkElement
	i.howManySmallLink = len(linkElement) - 1
	return i
}

//최종 H좌표, PH거리 계산 함수 (input : 1 Link(big), 1 Target(P))
//참조링크 : http://www.movable-type.co.uk/scripts/latlong.html
func (r ResultOfFindPoint) CalculateNearest(inputLink InputLink, inputTarget Position) ResultOfFindPoint {
	r.linkId = inputLink.linkId
	resultOfSmallLink := []ResultOfSmallLink{}
	for i := 0; i < inputLink.howManySmallLink; i++ {
		lon1 := inputLink.linkElement[i][0]
		lat1 := inputLink.linkElement[i][1]
		lon2 := inputLink.linkElement[i+1][0]
		lat2 := inputLink.linkElement[i+1][1]
		var tempResultOfSmallLink ResultOfSmallLink
		var tempLocation Position
		var positionOfH Position
		var PHdistance float64
		PHdistance = math.Abs(CalculateDistance_crossTrack(lon1, lat1, lon2, lat2, inputTarget.Longitude, inputTarget.Latitude))
		positionOfH, r.comeOrGoAway = tempLocation.CalculatePositionOfH(lon1, lat1, lon2, lat2, inputTarget.Longitude, inputTarget.Latitude)
		tempResultOfSmallLink.distance, tempResultOfSmallLink.isInLink = CheckWhereIsDestination(lon1, lat1, lon2, lat2, inputTarget.Longitude, inputTarget.Latitude, PHdistance, positionOfH, r.comeOrGoAway)
		if tempResultOfSmallLink.isInLink == "YES" {
			tempResultOfSmallLink.destination = positionOfH
		} else if tempResultOfSmallLink.isInLink == "NO" {
			if r.comeOrGoAway == "COME" {
				tempResultOfSmallLink.destination.Longitude = inputLink.linkElement[i+1][0]
				tempResultOfSmallLink.destination.Latitude = inputLink.linkElement[i+1][1]
			} else if r.comeOrGoAway == "GOAWAY" {
				tempResultOfSmallLink.destination.Longitude = inputLink.linkElement[i][0]
				tempResultOfSmallLink.destination.Latitude = inputLink.linkElement[i][1]
			}
		}
		resultOfSmallLink = append(resultOfSmallLink, tempResultOfSmallLink)
	}
	sort.Slice(resultOfSmallLink, func(i, j int) bool {
		return resultOfSmallLink[i].distance < resultOfSmallLink[j].distance
	})
	r.linkElement = inputLink.linkElement
	r.nearestDistance = resultOfSmallLink[0].distance
	r.nearestPosition.Latitude = resultOfSmallLink[0].destination.Latitude
	r.nearestPosition.Longitude = resultOfSmallLink[0].destination.Longitude
	r.isInLink = resultOfSmallLink[0].isInLink
	return r
}

//좌표 P와 링크(좌표A,B) 간 최단거리(PH) 계산 함수  (input : 3 Positions(P,A,B))
//cross track distance 계산 공식 활용 (given : 3 Positions(P,A,B))
func CalculateDistance_crossTrack(lon1 float64, lat1 float64, lon2 float64, lat2 float64, lon3 float64, lat3 float64) float64 {
	R := 6371000.00 //지구반지름 평균6371km
	dist13 := CalculateDistance_Harversine(lon1, lat1, lon3, lat3)
	brng13 := CalculateBearing(lon1, lat1, lon3, lat3) * math.Pi / 180
	brng12 := CalculateBearing(lon1, lat1, lon2, lat2) * math.Pi / 180
	delta13 := dist13 / R
	dXt := math.Asin(math.Sin(delta13)*math.Sin(brng13-brng12)) * R
	return dXt
}

//좌표 H(목적지)의 위도,경도 계산 함수 (input : 3 Positions(P,A,B), 1 Direction)
//Destination point 계산 공식 활용 (given : 1 Positions(A), 1 Bearing(AB), 1 Distance(AH))
func (H Position) CalculatePositionOfH(lon1 float64, lat1 float64, lon2 float64, lat2 float64, lon3 float64, lat3 float64) (Position, string) {
	R := 6371000.00 //지구반지름 평균6371km
	radLat1 := lat1 * math.Pi / 180
	var tempPosition1 Position
	var tempPosition2 Position
	var comeOrGoAway string
	brng12 := CalculateBearing(lon1, lat1, lon2, lat2) * math.Pi / 180
	tempdis := CalculateDistance_AlongTrack_FromStartPoint(lon1, lat1, lon2, lat2, lon3, lat3)
	dis := tempdis * 1.0
	aaaa := math.Asin(math.Sin(radLat1)*math.Cos(dis/R) + math.Cos(radLat1)*math.Sin(dis/R)*math.Cos(brng12))
	tempPosition1.Latitude = (aaaa*180/math.Pi + 360)
	if tempPosition1.Latitude > 360 {
		tempPosition1.Latitude -= 360
	}
	radPLat := tempPosition1.Latitude * math.Pi / 180
	bbbb := math.Atan2(math.Sin(brng12)*math.Sin(dis/R)*math.Cos(radLat1), math.Cos(dis/R)-math.Sin(radLat1)*math.Sin(radPLat))
	tempPosition1.Longitude = (bbbb*180/math.Pi + lon1 + 360)
	if tempPosition1.Longitude > 360 {
		tempPosition1.Longitude -= 360
	}
	dis = tempdis * -1.0
	aaaa = math.Asin(math.Sin(radLat1)*math.Cos(dis/R) + math.Cos(radLat1)*math.Sin(dis/R)*math.Cos(brng12))
	tempPosition2.Latitude = (aaaa*180/math.Pi + 360)
	if tempPosition2.Latitude > 360 {
		tempPosition2.Latitude -= 360
	}
	radPLat = tempPosition2.Latitude * math.Pi / 180
	bbbb = math.Atan2(math.Sin(brng12)*math.Sin(dis/R)*math.Cos(radLat1), math.Cos(dis/R)-math.Sin(radLat1)*math.Sin(radPLat))
	tempPosition2.Longitude = (bbbb*180/math.Pi + lon1 + 360)
	if tempPosition2.Longitude > 360 {
		tempPosition2.Longitude -= 360
	}
	distPH1 := math.Abs(CalculateDistance_Harversine(tempPosition1.Longitude, tempPosition1.Latitude, lon3, lat3))
	distPH2 := math.Abs(CalculateDistance_Harversine(tempPosition2.Longitude, tempPosition2.Latitude, lon3, lat3))
	if distPH1 <= distPH2 {
		H.Latitude = tempPosition1.Latitude
		H.Longitude = tempPosition1.Longitude
		comeOrGoAway = "COME"
	} else {
		H.Latitude = tempPosition2.Latitude
		H.Longitude = tempPosition2.Longitude
		comeOrGoAway = "GOAWAY"
	}
	return H, comeOrGoAway
}

//좌표 A와 목적지 H 간 거리(AH) 계산 함수 (input : 3 Positions(P,A,B))
//along track distance 계산 공식 활용 (given : 1 Positions(A),2 Distance(AP, PH))
func CalculateDistance_AlongTrack_FromStartPoint(lon1 float64, lat1 float64, lon2 float64, lat2 float64, lon3 float64, lat3 float64) float64 {
	R := 6371000.00 //지구반지름 평균6371km
	dist13 := CalculateDistance_Harversine(lon1, lat1, lon3, lat3)
	delta13 := dist13 / R
	dXt := CalculateDistance_crossTrack(lon1, lat1, lon2, lat2, lon3, lat3)
	dSt := math.Acos(math.Cos(delta13)/math.Cos(dXt/R)) * R
	return dSt
}

//두 좌표간 Bearing 계산 함수  (input : 2 Positions)
//Bearing(방위각) : 좌표A→B진행방향을 A점에서 북쪽을 기준으로 구한 각도 (given : 2 Positions)
func CalculateBearing(lon1 float64, lat1 float64, lon2 float64, lat2 float64) float64 {
	radLon1 := lon1 * math.Pi / 180
	radLon2 := lon2 * math.Pi / 180
	radLat1 := lat1 * math.Pi / 180
	radLat2 := lat2 * math.Pi / 180
	radDelLon := (lon2 - lon1) * math.Pi / 180
	y := math.Sin(radLon2-radLon1) * math.Cos(radLat2)
	x := math.Cos(radLat1)*math.Sin(radLat2) - math.Sin(radLat1)*math.Cos(radLat2)*math.Cos(radDelLon)
	th := math.Atan2(y, x)
	brng := (th*180/math.Pi + 360)
	if brng > 360 {
		brng -= 360
	}
	return brng
}

//두 좌표간 거리 계산 함수 (input : 2 Positions)
//Harversine : 지도상에서 두개의 위경도 좌표간의 거리 계산 공식(given : 2 Positions)
func CalculateDistance_Harversine(lon1 float64, lat1 float64, lon2 float64, lat2 float64) float64 {
	R := 6371000.00 //지구반지름 평균6371km
	radLat1 := lat1 * math.Pi / 180
	radLat2 := lat2 * math.Pi / 180
	radDelLat := (lat2 - lat1) * math.Pi / 180
	radDelLon := (lon2 - lon1) * math.Pi / 180
	a := math.Pow(math.Sin(radDelLat/2), 2) + math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(radDelLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c
	return d
}

//Target(P)으로부터 링크 방향, 최단거리 도출 함수 (input : 3 Positions(P,A,B))
//직접 구현 함수(가정 : 좌표A → 좌표B 방향 존재)) (given : 3 Positions(P,A,B))
func CheckComeOrGoAway(lon1 float64, lat1 float64, lon2 float64, lat2 float64, lon3 float64, lat3 float64) string {
	distPA := math.Abs(CalculateDistance_Harversine(lon1, lat1, lon3, lat3))
	distPB := math.Abs(CalculateDistance_Harversine(lon2, lat2, lon3, lat3))
	direction := ""
	if distPA >= distPB {
		direction = "COME"
	} else if distPB >= distPA {
		direction = "GOAWAY"
	}
	return direction
}

//Target(P)으로부터 링크 방향, 최단거리 도출 함수 (input : 4 Positions(P,A,B,H), 1 Distance(PH), 1 Direction(P기준 A→B방향))
//직접 구현 함수(가정 : 좌표A → 좌표B 방향 존재)) (given : 3 Positions(P,A,B))
func CheckWhereIsDestination(lon1 float64, lat1 float64, lon2 float64, lat2 float64, lon3 float64, lat3 float64, distPH float64, H Position, Direction string) (float64, string) {
	isInLink := ""
	nearestDistance := 0.0
	if ((lon1 <= H.Longitude && H.Longitude <= lon2) || (lon2 <= H.Longitude && H.Longitude <= lon1)) && ((lat1 <= H.Latitude && H.Latitude <= lat2) || (lat2 <= H.Latitude && H.Latitude <= lat1)) {
		isInLink = "YES"
		nearestDistance = distPH
	} else {
		isInLink = "NO"
		if Direction == "COME" {
			nearestDistance = math.Abs(CalculateDistance_Harversine(lon2, lat2, lon3, lat3))
		} else if Direction == "GOAWAY" {
			nearestDistance = math.Abs(CalculateDistance_Harversine(lon1, lat1, lon3, lat3))
		}
	}
	return nearestDistance, isInLink
}

//파일 read시 에러 보정 함수
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	//Arguments 파싱하기
	args := os.Args                           ////Arguments 전체
	restArgs := args[1:]                      //0번째 Arguments 인자를 제외한 나머지 Arguments
	inputFile := restArgs[2]                  //2번째 Arguments 인자 : 로딩할 input파일명(geojson) (findpoint.go와 동일 폴더내에 위치해야함)
	target := strings.Split(restArgs[4], ",") //4번째 Arguments 인자 : target(좌표P) 추출

	//input 파일 read(.geojson)
	dat, err := ioutil.ReadFile(inputFile)
	check(err)

	//input 파일 geojson→GO 변환
	rawFeatureJSON := []byte(dat)
	fc1, _ := geojson.UnmarshalFeatureCollection(rawFeatureJSON)

	//Arguments로 파싱한 target(좌표P)의 형변환(str→float64)
	var inputTarget Position
	inputTarget.Longitude, _ = strconv.ParseFloat(target[0], 64)
	inputTarget.Latitude, _ = strconv.ParseFloat(target[1], 64)

	inputSlice := []InputLink{}          //슬라이스선언 : input으로 들어온 링크들을 id별 InputLink구조체의 slice
	resultSlice := []ResultOfFindPoint{} //슬라이스선언 : id별 계산된 Link와 좌표P간의 최단거리정보(ResultOfFindPoint) 구조체의 slice

	//Link와 좌표P간의 최단거리 계산(inputSlice에서 id별 계산 후 resultSlice에 append)
	for i := 0; i < len(fc1.Features); i++ {
		var tempInput InputLink                                   //계산에 임시로 활용될 id별 InputLink구조체 선언
		var tempResultOfFind ResultOfFindPoint                    //계산결과를 임시로 저장할 id별 ResultOfFindPoint구조체 선언
		idtmp, ok := (*fc1.Features[i]).Properties["id"].(string) //input으로 들어온 링크의 id 저장
		//input값이 잘 들어왔을 시
		if ok {
			linkInfo := tempInput.GetLink(idtmp, (*fc1.Features[i]).Geometry.LineString) //id별 input으로 들어온 링크의 정보 추출
			inputSlice = append(inputSlice, linkInfo)                                    //id별 input으로 들어온 링크의 정보 Slice에 저장
			resultOfFindInfo := tempResultOfFind.CalculateNearest(linkInfo, inputTarget) //id별 계산된 Link와 좌표P간의 최단거리정보 계산
			resultSlice = append(resultSlice, resultOfFindInfo)                          //id별 계산된 Link와 좌표P간의 최단거리정보  Slice에 저장
		}
	}

	//최종 결과 구조체의 슬라이스를 sorting(최단거리 순서)
	sort.Slice(resultSlice, func(i, j int) bool {
		return resultSlice[i].nearestDistance < resultSlice[j].nearestDistance
	})

	//최종 최단거리 출력
	fmt.Println("가까운 좌표까지 거리 미터,가까운 좌표의 경도,가까운 좌표의 위도  ")
	fmt.Print(resultSlice[0].nearestDistance)
	fmt.Print(",")
	fmt.Print(resultSlice[0].nearestPosition.Longitude)
	fmt.Print(",")
	fmt.Println(resultSlice[0].nearestPosition.Latitude)
}

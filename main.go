package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/dhconnelly/rtreego"
	"github.com/hailocab/go-geoindex"
)

type (
	Shop struct {
		Id_  string  `json:"id"`
		Lat_ float64 `json:"geo_lat"`
		Lon_ float64 `json:"geo_lng"`
	}
)

/*
 30.324462,50.247284,30.858672,50.587729
 bL 50.247284,30.324462	tR 50.587729,30.858672
 tL 50.587729, 30.324462	bR 50.247284, 30.858672
*/

func main() {
	defer testTime("main", time.Now())

	if len(os.Args) != 2 {
		panicIfError(fmt.Errorf("required URL does not exists"))
	}

	shops := makeSliceFromShops(os.Args[1])
	fmt.Println(len(shops))
	testLibGeoIndex(shops)
	testLibRTree(shops)

	//for _, v := range points {
	//	fmt.Println(v.Id())
	//}

	var p1, p2 *xPoint
	p1 = &xPoint{50.425365, 30.459593}
	p2 = &xPoint{50.4214319750507, 30.458242893219}
	fmt.Println(GreatCircleDistance(p1, p2))
	p1 = &xPoint{50.425365, 30.459593}
	p2 = &xPoint{50.422747, 30.464512}
	fmt.Println(GreatCircleDistance(p1, p2))
}

func makeSliceFromShops(url string) []*Shop {
	shops := make([]*Shop, 0, 10000)

	res, err := http.Get(os.Args[1])
	panicIfError(err)
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&shops)
	panicIfError(err)

	return shops
}

func testLibGeoIndex(shops []*Shop) {
	index := makeIndex1(shops)

	var points []geoindex.Point
	p := &geoindex.GeoPoint{"", 50.425365, 30.459593}
	points = testKNearest1(index, p, 100, geoindex.Meters(1000000))
	fmt.Println(len(points))

	for _, v := range points {
		fmt.Println(v.Id())
	}

	p1 := &geoindex.GeoPoint{"tL", 50.587729, 30.324462}
	p2 := &geoindex.GeoPoint{"bR", 50.247284, 30.858672}
	points = testRange1(index, p1, p2)
	fmt.Println(len(points))
}

func testLibRTree(shops []*Shop) {
	index := makeIndex2(shops)

	var points []rtreego.Spatial
	p := rtreego.Point{50.425365, 30.459593}
	points = testKNearest2(index, p, 5)
	fmt.Println(len(points))

	for _, v := range points {
		fmt.Println(v.(*Shop).Id_)
	}

	p1 := rtreego.Point{50.587729, 30.324462}
	p2 := rtreego.Point{50.247284, 30.858672}
	points = testRange2(index, p1, p2)
	fmt.Println(len(points))
}

func makeIndex1(shops []*Shop) *geoindex.PointsIndex {
	defer testTime("index 1", time.Now())
	index := geoindex.NewPointsIndex(geoindex.Km(1.5))
	for _, v := range shops {
		if v.Lat() == 0 && v.Lon() == 0 {
			continue
		}
		index.Add(v)
	}
	return index
}

func testKNearest1(index *geoindex.PointsIndex, p *geoindex.GeoPoint, k int, d geoindex.Meters) []geoindex.Point {
	defer testTime("knearest 1", time.Now())
	return index.KNearest(p, k, d, func(p geoindex.Point) bool { return true })
}

func testRange1(index *geoindex.PointsIndex, p1, p2 *geoindex.GeoPoint) []geoindex.Point {
	defer testTime("range 1", time.Now())
	return index.Range(p1, p2)
}

func makeIndex2(shops []*Shop) *rtreego.Rtree {
	defer testTime("index 2", time.Now())
	index := rtreego.NewTree(2, 25, 50)
	for _, v := range shops {
		if v.Lat() == 0 && v.Lon() == 0 {
			continue
		}
		index.Insert(v)
	}
	return index
}

func testKNearest2(index *rtreego.Rtree, p rtreego.Point, k int) []rtreego.Spatial {
	defer testTime("knearest 2", time.Now())
	return index.NearestNeighbors(k, p)
}

func testRange2(index *rtreego.Rtree, p1, p2 rtreego.Point) []rtreego.Spatial {
	defer testTime("range 2", time.Now())
	bb, _ := rtreego.NewRect(p2, []float64{p1[0], p1[1]})
	return index.SearchIntersect(bb)
}

// Implement geoindex.Point interface
func (s *Shop) Id() string {
	return s.Id_
}

func (s *Shop) Lat() float64 {
	return s.Lat_
}

func (s *Shop) Lon() float64 {
	return s.Lon_
}

// Implement rtreego.Spatial interface
func (s *Shop) Bounds() *rtreego.Rect {
	return rtreego.Point{s.Lat(), s.Lon()}.ToRect(0.01)
}

// Represents a Physical Point in geographic notation [lat, lng].
type xPoint struct {
	Lat float64
	Lng float64
}

// Calculates the Haversine distance between two points.
// Original Implementation from: http://www.movable-type.co.uk/scripts/latlong.html
// https://github.com/kellydunn/golang-geo/blob/develop/point.go
func GreatCircleDistance(p1, p2 *xPoint) float64 {
	// According to Wikipedia, the Earth's radius is about 6,371km
	const EARTH_RADIUS = 6371.21

	dLat := (p2.Lat - p1.Lat) * (math.Pi / 180.0)
	dLng := (p2.Lng - p1.Lng) * (math.Pi / 180.0)

	lat1 := p1.Lat * (math.Pi / 180.0)
	lat2 := p2.Lat * (math.Pi / 180.0)

	a1 := math.Sin(dLat/2) * math.Sin(dLat/2)
	a2 := math.Sin(dLng/2) * math.Sin(dLng/2) * math.Cos(lat1) * math.Cos(lat2)

	a := a1 + a2

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EARTH_RADIUS * c
}

func testTime(s string, t time.Time) {
	fmt.Printf("[%s] elapsed time: %s\n", s, time.Since(t))
}

// -------------------------

func panicIfError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func test(v string, accept func(v string) bool) {
	if accept == nil || accept(v) {
		fmt.Println(v)
	}
}

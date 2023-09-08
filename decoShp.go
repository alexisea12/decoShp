package decoshp

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"math"
)

var shapeTypes = map[int]string{
	0:	"Null Shape",
	1:	"Point",
	3:	"PolyLine",
	5:	"Polygon",
	8:	"MultiPoint",
	11:	"PointZ",
	13:	"PolyLineZ",
	15:	"PolygonZ",
	18:	"MultiPointZ",
	21:	"PointM",
	23:	"PolyLineM",
	25:	"PolygonM",
	28:	"MultiPointM",
	31:	"MultiPatch",
}

type Point struct{
	X	float64
	Y	float64
}

type PolygonBoundingBox struct {
	XMin	float64
	YMin	float64
	XMax	float64
	YMax	float64
}

type PolygonRecord struct {
	Header	struct{
		RecordNumber	int
		ContentLenght	int 
	}
	ShapeType	int
	Box			PolygonBoundingBox
	NumParts	int
	NumPoints	int
	Parts		[]int
	Points		[]Point 
}


type BoundingBox struct {
	Xmin, Ymin, Xmax, Ymax, Zmin, Zmax, Mmin, Mmax float64
}

type Header struct {
	FileCode	int
	FileLength	int
	Version		int
	ShapeType	int
	BoundingBox
}

type Decoder struct {
	file *os.File 
	*Header 
	bytesRead	int
}

func New(file *os.File) (*Decoder, error) {
	d := &Decoder{file: file}
	h, err := d.GetHeader()
	if err != nil {
		return nil, err 
	}
	d.Header = h 
	return d, nil
}

func(d *Decoder) GetHeader() (*Header, error){
	var currHeader Header 
	b1 := make([]byte, 100)
//	r4 := bufio.NewReader(d.file)
	_, err := io.ReadFull(d.file, b1)

	if err != nil {
		return nil, fmt.Errorf("error while reading file: %e", err)
	}
	
	d.bytesRead += 50

	code := binary.BigEndian.Uint32(b1[:4])
	fileLenght := binary.BigEndian.Uint32(b1[24:28])
	version := binary.LittleEndian.Uint16(b1[28:32])
	shapeType := binary.LittleEndian.Uint16(b1[32:36])

	xMin := binary.LittleEndian.Uint64(b1[36:44])
	yMin := binary.LittleEndian.Uint64(b1[44:52])
	xMax := binary.LittleEndian.Uint64(b1[52:60])
	yMax := binary.LittleEndian.Uint64(b1[60:68])
	zMin := binary.LittleEndian.Uint64(b1[68:76])
	zMax := binary.LittleEndian.Uint64(b1[76:84])
	mMin := binary.LittleEndian.Uint64(b1[84:92])
	mMax := binary.LittleEndian.Uint64(b1[92:])

	box := BoundingBox{
		Xmin: math.Float64frombits(xMin),
		Ymin: math.Float64frombits(yMin),
		Xmax: math.Float64frombits(xMax),
		Ymax: math.Float64frombits(yMax),
		Zmin: math.Float64frombits(zMin),
		Zmax: math.Float64frombits(zMax),
		Mmin: math.Float64frombits(mMin),
		Mmax: math.Float64frombits(mMax),
	}

	currHeader = Header{int(code), int(fileLenght), int(version), int(shapeType), box}

	return &currHeader, nil 
}

func (d *Decoder) DecodeRecord()(*PolygonRecord, error){
	if d.bytesRead >= d.Header.FileLength {
		return nil, io.EOF
	}
	var decodedRecord PolygonRecord
	header := make([]byte, 8)

	_, err := io.ReadFull(d.file, header)
	if err != nil {
		return nil, err 
	}

	recordNumber := binary.BigEndian.Uint32(header[:4])
	recordLenght := binary.BigEndian.Uint32(header[4:])
	// we set the header
	decodedRecord.Header.RecordNumber = int(recordNumber)
	decodedRecord.Header.ContentLenght = int(recordLenght)

	recordBody := make([]byte, recordLenght*2)
	_, err = io.ReadFull(d.file, recordBody)
	if err != nil {
		return nil, err 
	}

	d.bytesRead += int(recordLenght)+4

	shapeType := binary.LittleEndian.Uint32(recordBody[:4])
	decodedRecord.ShapeType = int(shapeType)
	// Decode the boundind box
	boxXMin := binary.LittleEndian.Uint64(recordBody[4:4+8])
	boxYMin := binary.LittleEndian.Uint64(recordBody[4+8:4+16])
	boxXMax := binary.LittleEndian.Uint64(recordBody[4+16:4+24])
	boxYMax := binary.LittleEndian.Uint64(recordBody[4+24:4+32])
	decodedRecord.Box = PolygonBoundingBox{
		XMin: math.Float64frombits(boxXMin),
		YMin: math.Float64frombits(boxYMin),
		XMax: math.Float64frombits(boxXMax),
		YMax: math.Float64frombits(boxYMax),
	}

	numParts := binary.LittleEndian.Uint32(recordBody[36:40])
	decodedRecord.NumParts = int(numParts)

	numPoints := binary.LittleEndian.Uint32(recordBody[40:44])
	decodedRecord.NumPoints = int(numPoints)

	var partsArray []int 
	parts := recordBody[44:44+4*numParts]
	for i := 0; i < len(parts); i+=4 {
		val := binary.LittleEndian.Uint32(parts[i:i+4])
		partsArray = append(partsArray, int(val))
	}

	points := recordBody[44+4*numParts:]
	
	decodedPoints := DecodePoints(points, int(recordLenght))
	decodedRecord.Points = decodedPoints

	return &decodedRecord, nil 
}


func DecodePoints(bytesArray []byte, lenght int) []Point {
	var pointsArray []Point
	for i := 0; i < len(bytesArray); i+=16 {
		var point Point
		x := binary.LittleEndian.Uint64(bytesArray[i:i+8])
		y := binary.LittleEndian.Uint64(bytesArray[i+8:i+16])
		point.X = math.Float64frombits(x) *13 + 1500
		point.Y = math.Float64frombits(y) *13 - 400
		pointsArray = append(pointsArray, point)
	}
	return pointsArray
}



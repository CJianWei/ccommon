package db

// @Time    : 2018/3/26 16:41
// @Author  : chenjw
// @Site    :
// @File    : influx.go
// @Software: GoLand
import (
	"github.com/Centny/gwf/log"
	"github.com/influxdata/influxdb/client/v2"
	"sync"
	"time"
)

type Influx struct {
	Addr      string
	Username  string
	Password  string
	Database  string
	Precision string
	C         func() client.Client
	signal    chan *client.Point
	lock      *sync.Mutex
}

func NewInflux(addr, username, password, database, precision string, SignalSize int) *Influx {
	In := &Influx{
		Addr:      addr,
		Username:  username,
		Password:  password,
		Database:  database,
		Precision: precision,
		lock:      &sync.Mutex{},
		signal:    make(chan *client.Point, SignalSize),
	}
	go In.syncData()
	return In
}

func NewInflux2(addr, database string) *Influx {
	return NewInflux(addr, "", "", database, "ns", 1<<15)
}

func (In *Influx) AddUsername(Username string) *Influx {
	In.Username = Username
	return In
}

func (In *Influx) AddPassword(Password string) *Influx {
	In.Password = Password
	return In
}

func (In *Influx) InitDb() error {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     In.Addr,
		Username: In.Username,
		Password: In.Password,
	})
	if err != nil {
		log.E("[Influx] [InitDb] err %v", err.Error())
		return err
	}
	In.C = func() client.Client {
		return c
	}
	return nil
}

func (In *Influx) Tick() {
	for {
		log.I("[Influx] [tick] ...")
		time.Sleep(time.Second * 3)
		_, _, err := In.C().Ping(time.Second * 1)
		if err != nil {
			In.InitDb()
		}
	}
}

func (In *Influx) newBp() (client.BatchPoints, error) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  In.Database,
		Precision: In.Precision,
	})
	if err != nil {
		log.E("[Influx] [newBp] err %v", err.Error())
	}
	return bp, err
}

func (In *Influx) newPt(name string, tags map[string]string, fields map[string]interface{}) (*client.Point, error) {
	pt, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		log.E("[Influx] [newPt] err %v", err.Error())
	}
	return pt, err
}

func (In *Influx) write(bp client.BatchPoints) error {
	err := In.C().Write(bp)
	if err != nil {
		log.E("[Influx] [write] err %v", err.Error())
	} else {
		//log.D("[Influx] [write] write data count %v", len(bp.Points()))
	}
	return err
}

//add single point
func (In *Influx) AddPoint(name string, tags map[string]string, fields map[string]interface{}) error {
	bp, err := In.newBp()
	if err != nil {
		return err
	}
	pt, err := In.newPt(name, tags, fields)
	if err != nil {
		return err
	}
	bp.AddPoint(pt)
	return In.write(bp)
}

//add multi points
func (In *Influx) AddPoints(pts []*client.Point) error {
	bp, err := In.newBp()
	if err != nil {
		return err
	}
	bp.AddPoints(pts)
	return In.write(bp)
}

//add single points by sync
func (In *Influx) AddPointSync(name string, tags map[string]string, fields map[string]interface{}) error {
	pt, err := In.newPt(name, tags, fields)
	if err != nil {
		return err
	}
	In.signal <- pt
	return nil
}

//add multi points by sync
func (In *Influx) AddPointsSync(pts []*client.Point) error {
	for _, pt := range pts {
		In.signal <- pt
	}
	return nil
}

func (In *Influx) writeToDb() {
	In.lock.Lock()
	defer In.lock.Unlock()
	pts := []*client.Point{}
	for i := 0; i < 50; i++ {
		if len(In.signal) > 0 {
			pts = append(pts, <-In.signal)
		} else {
			break
		}
	}
	if len(pts) > 0 {
		//log.D("[Influx] [writeToDb] size %v", len(pts))
		In.AddPoints(pts)
	}
}

func (In *Influx) syncData() error {
	for {
		time.Sleep(time.Millisecond)
		In.writeToDb()
	}
}

func (In *Influx) Flush() error {
	for {
		if len(In.signal) > 0 {
			In.writeToDb()
		} else {
			break
		}
	}
	return nil
}

func (In *Influx) Close() {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	In.C().Close()
}

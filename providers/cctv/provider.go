package cctv

import (
	"fmt"
	"time"
	wheretopark "wheretopark/go"

	"gocv.io/x/gocv"
)

type Provider struct {
	configuration Configuration
	model         *Model
	streams       [][]*gocv.VideoCapture
	savePath      *string
}

func (p Provider) GetMetadata() (map[wheretopark.ID]wheretopark.Metadata, error) {
	metadatas := make(map[wheretopark.ID]wheretopark.Metadata, len(p.configuration.ParkingLots))

	for _, parkingLot := range p.configuration.ParkingLots {
		id := wheretopark.GeometryToID(parkingLot.Geometry)
		metadatas[id] = parkingLot.Metadata
	}

	fmt.Printf("obtained %d metadatas\n", len(metadatas))
	return metadatas, nil
}

func (p Provider) GetState() (map[wheretopark.ID]wheretopark.State, error) {
	states := make(map[wheretopark.ID]wheretopark.State)

	img := gocv.NewMat()
	for i, parkingLot := range p.configuration.ParkingLots {
		availableSpots := 0
		captureTime := time.Now()
		for k, camera := range parkingLot.Cameras {
			fmt.Printf("processing %s/%d\n", parkingLot.Name, k)
			stream := p.streams[i][k]
			if ok := stream.Read(&img); !ok {
				return nil, fmt.Errorf("cannot read stream of %s", parkingLot.Name)
			}
			spotImages := ExtractSpots(img, camera.Spots)
			predictions := p.model.PredictMany(spotImages)
			for _, prediction := range predictions {
				if prediction > 0.5 {
					availableSpots += 1
				}
			}

			if p.savePath != nil {
				basePath := fmt.Sprintf("%s/%s/%s/%02d", *p.savePath, parkingLot.Name, captureTime.UTC().Format("2006-01-02--15-04-05"), k+1)
				SavePredictions(img, basePath, camera.Spots, predictions)
			}
		}
		id := wheretopark.GeometryToID(parkingLot.Geometry)
		state := wheretopark.State{
			LastUpdated: time.Now().Format(time.RFC3339),
			AvailableSpots: map[string]uint{
				"CAR": uint(availableSpots),
			},
		}
		states[id] = state
	}
	fmt.Printf("obtained %d states\n", len(states))
	return states, nil
}

func (p Provider) Close() error {
	for _, streams := range p.streams {
		for _, stream := range streams {
			if err := stream.Close(); err != nil {
				return err
			}
		}

	}
	return nil
}

func NewProvider(configurationPath *string, model *Model, savePath *string) (*Provider, error) {
	var configuration Configuration
	if configurationPath == nil {
		configuration = DefaultConfiguration
	} else {
		newConfiguration, err := LoadConfiguration(*configurationPath)
		if err != nil {
			return nil, err
		}
		configuration = *newConfiguration
	}

	streams := make([][]*gocv.VideoCapture, len(configuration.ParkingLots))
	for i, parkingLot := range configuration.ParkingLots {
		streams[i] = make([]*gocv.VideoCapture, len(parkingLot.Cameras))
		for k, camera := range parkingLot.Cameras {
			fmt.Printf("connecting to %s\n", camera.URL)
			stream, err := gocv.OpenVideoCapture(camera.URL)
			if err != nil {
				return nil, err
			}
			fmt.Printf("connected\n")
			streams[i][k] = stream
		}
	}

	return &Provider{
		configuration: configuration,
		model:         model,
		streams:       streams,
		savePath:      savePath,
	}, nil

}

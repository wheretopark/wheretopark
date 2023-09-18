package cctv

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	wheretopark "wheretopark/go"

	"github.com/rs/zerolog/log"
)

type Source struct {
	id       wheretopark.ID
	metadata wheretopark.Metadata
	cameras  []ParkingLotCamera
	model    Model
	saver    Saver
}

func (s *Source) Metadata(context.Context) (map[wheretopark.ID]wheretopark.Metadata, error) {
	return map[wheretopark.ID]wheretopark.Metadata{
		s.id: s.metadata,
	}, nil
}

func (s *Source) State(ctx context.Context) (map[wheretopark.ID]wheretopark.State, error) {
	availableSpotsPtr := make(map[wheretopark.SpotType]*uint32, len(wheretopark.SpotTypes))
	for _, spotType := range wheretopark.SpotTypes {
		availableSpotsPtr[spotType] = new(uint32)
	}

	var wg sync.WaitGroup
	captureTime := time.Now()
	for id, camera := range s.cameras {
		wg.Add(1)
		log.Ctx(ctx).
			Info().
			Int("camera", id).
			Msg("processing parking lot camera")
		go func(id int, camera ParkingLotCamera) {
			defer wg.Done()
			camAvailableSpots, err := s.processCamera(camera)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Int("id", id).Msg("processing camera fail")
				return
			}
			for spotType, count := range camAvailableSpots {
				atomic.AddUint32(availableSpotsPtr[spotType], uint32(count))
			}
		}(id, camera)
	}
	wg.Wait()
	log.Ctx(ctx).
		Info().
		Str("duration", time.Since(captureTime).String()).
		Msg("finished processing cameras")

	availableSpots := make(map[wheretopark.SpotType]uint)
	// TODO: Make clients to comply with missing spot type car
	availableSpots[wheretopark.SpotTypeCar] = 0
	for spotType, countPtr := range availableSpotsPtr {
		if *countPtr > 0 {
			availableSpots[spotType] = uint(*countPtr)
		}
	}
	state := wheretopark.State{
		LastUpdated:    captureTime,
		AvailableSpots: availableSpots,
	}
	return map[wheretopark.ID]wheretopark.State{
		s.id: state,
	}, nil
}

func (s *Source) processCamera(camera ParkingLotCamera) (map[wheretopark.SpotType]uint, error) {
	img, err := GetImageFromCamera(camera.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to get image from camera: %v", err)
	}
	defer img.Close()

	spotImages := ExtractSpots(img, camera.Spots)
	defer func() {
		for _, spotImage := range spotImages {
			spotImage.Close()
		}
	}()
	predictions := s.model.PredictMany(spotImages)

	availableSpots := map[wheretopark.SpotType]uint{}
	for i, prediction := range predictions {
		if IsVacant(prediction) {
			spotType := camera.Spots[i].Type
			availableSpots[spotType]++
		}
	}

	// err = p.saver.SavePredictions(img, id, cameraID, captureTime, camera.Spots, predictions)
	// if err != nil {
	// 	log.Error().
	// 		Str("name", parkingLot.Name).
	// 		Int("camera", cameraID).
	// 		Msg("unable to save predictions")
	// }
	return availableSpots, nil
}

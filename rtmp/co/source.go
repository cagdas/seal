package co

import (
	"log"
	"seal/conf"
	"seal/hls"
	"seal/rtmp/pt"
	"sync"
)

// stream data source hub
type sourceHub struct {
	//key: app/streamName, e.g. rtmp://127.0.0.1/live/test, the key is [live/app]
	hub  map[string]*SourceStream
	lock sync.RWMutex
}

var gSources = &sourceHub{
	hub: make(map[string]*SourceStream),
}

// SourceStream rtmp stream data source
type SourceStream struct {
	// the sample rate of audio in metadata
	sampleRate float64
	// the video frame rate in metadata
	frameRate float64
	// atc whether atc(use absolute time and donot adjust time),
	// directly use msg time and donot adjust if atc is true,
	// otherwise, adjust msg time to start from 0 to make flash happy.
	atc bool
	// time jitter algrithem
	timeJitter uint32
	// cached meta data
	cacheMetaData *pt.Message
	// cached video sequence header
	cacheVideoSequenceHeader *pt.Message
	// cached aideo sequence header
	cacheAudioSequenceHeader *pt.Message

	// consumers
	consumers map[*Consumer]interface{}
	// lock for consumers.
	consumerLock sync.RWMutex

	// gop cache
	gopCache *GopCache

	// hls stream
	hls *hls.SourceStream
}

func (s *SourceStream) createConsumer(c *Consumer) {
	if nil == c {
		log.Println("when registe consumer, nil == consumer")
		return
	}

	s.consumerLock.Lock()
	defer s.consumerLock.Unlock()

	s.consumers[c] = struct{}{}
	log.Println("a consumer created.consumer=", c)

}

func (s *SourceStream) DestroyConsumer(c *Consumer) {
	if nil == c {
		log.Println("when destroy consumer, nil == consummer")
		return
	}

	s.consumerLock.Lock()
	defer s.consumerLock.Unlock()

	delete(s.consumers, c)
	log.Println("a consumer destroyed.consumer=", c)
}

func (s *SourceStream) copyToAllConsumers(msg *pt.Message) {

	if nil == msg {
		return
	}

	s.consumerLock.Lock()
	defer s.consumerLock.Unlock()

	for k, v := range s.consumers {
		_ = v
		k.enquene(msg, s.atc, s.sampleRate, s.frameRate, s.timeJitter)
	}
}

func (s *sourceHub) findSourceToPublish(k string) *SourceStream {

	if 0 == len(k) {
		log.Println("find source to publish, nil == k")
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if res := s.hub[k]; nil != res {
		log.Println("stream ", k, " can not publish, because has already publishing....")
		return nil
	}

	//can publish. new a source
	s.hub[k] = &SourceStream{
		timeJitter: conf.GlobalConfInfo.Rtmp.TimeJitter,
		gopCache:   &GopCache{},
		consumers:  make(map[*Consumer]interface{}),
	}

	if "true" == conf.GlobalConfInfo.Hls.Enable {
		s.hub[k].hls = hls.NewSourceStream()
	} else {
		// make sure is nil when hls is closed
		s.hub[k].hls = nil
	}

	return s.hub[k]
}

func (s *sourceHub) findSourceToPlay(k string) *SourceStream {
	s.lock.Lock()
	defer s.lock.Unlock()

	if res := s.hub[k]; nil != res {
		return res
	}

	log.Println("stream ", k, " can not play, because has not published.")

	return nil
}

func (s *sourceHub) deleteSource(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	stream := s.hub[key]
	if nil != stream {
		stream.hls.OnUnPublish()
	}

	delete(s.hub, key)
}

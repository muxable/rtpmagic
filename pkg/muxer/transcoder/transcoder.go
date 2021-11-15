package transcoder

type Transcoder struct {
	mimeType string
}

func NewTranscoder(mimeType string) (*Transcoder) {
	t := &Transcoder{
		mimeType: mimeType,
	}
	go t.start()
	return t
}

func (t *Transcoder) start() {
	pipeline := gstreamer.NewPipeline(`
		appsrc name=appsrc caps="video/x-raw,format=I420" is-live=true ! decodebin ! videoconvert ! autovideosink
	`)
	for {
		n, err := 
	}

func (t *Transcoder) Read(data []byte) ([]byte, error) {

}

func (t *Transcoder) Write(data []byte) ([]byte, error) {

}

func (t *Transcoder) Close() error {
	
}
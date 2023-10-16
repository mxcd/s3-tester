package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/mxcd/s3-tester/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "s3-tester",
		Usage: "S3 Tester",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "debug output",
				EnvVars: []string{"S3_VERBOSE"},
			},
			&cli.BoolFlag{
				Name:    "very-verbose",
				Aliases: []string{"vv"},
				Usage:   "trace output",
				EnvVars: []string{"S3_VERY_VERBOSE"},
			},
			&cli.StringFlag{
				Name:    "endpoint",
				Aliases: []string{"e"},
				Usage:   "s3 endpoint",
				EnvVars: []string{"S3_ENDPOINT"},
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "s3 port",
				EnvVars: []string{"S3_PORT"},
			},
			&cli.StringFlag{
				Name:    "access-key",
				Aliases: []string{"a"},
				Usage:   "s3 access key",
				EnvVars: []string{"S3_ACCESS_KEY"},
			},
			&cli.StringFlag{
				Name:    "secret-key",
				Aliases: []string{"s"},
				Usage:   "s3 secret key",
				EnvVars: []string{"S3_SECRET_KEY"},
			},
			&cli.StringFlag{
				Name:    "bucket",
				Aliases: []string{"b"},
				Usage:   "s3 bucket",
				EnvVars: []string{"S3_BUCKET"},
			},
			&cli.BoolFlag{
				Name:    "insecure",
				Usage:   "s3 insecure connection",
				EnvVars: []string{"S3_INSECURE"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "upload",
				Aliases: []string{"u"},
				Usage:   "Upload a file to the specified S3 bucket",
				Action: func(c *cli.Context) error {
					initLogger(c)
					return upload(c)
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"r"},
				Usage:   "Remove a file from the specified S3 bucket",
				Action: func(c *cli.Context) error {
					initLogger(c)
					return remove(c)
				},
			},
			{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "Generates a pre-signed URL for the specified S3 object",
				Action: func(c *cli.Context) error {
					initLogger(c)
					return sign(c)
				},
			},
			{
				Name:  "performance",
				Usage: "Tests S3 performance. s3-tester performance",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "vus",
						Usage: "Virtual users",
					},
					&cli.IntFlag{
						Name:  "duration",
						Usage: "Duration in seconds",
					},
					&cli.StringFlag{
						Name:  "filesize",
						Usage: "File size in bytes",
					},
				},
				Action: func(c *cli.Context) error {
					initLogger(c)
					return performance(c)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err)
	}
}

func upload(c *cli.Context) error {
	if c.Args().Len() != 1 {
		log.Fatal().Msg("Please specify a file to upload")
	}
	filePath := c.Args().First()
	id := uuid.New().String()

	// check if file exists
	stats, err := os.Stat(filePath)
	if err != nil {
		log.Fatal().Err(err).Msgf("File '%s' does not exist", filePath)
	}
	fileSize := stats.Size()

	log.Info().Msgf("File '%s' exists with size '%s'", filePath, util.GetStringFromByteSize(fileSize))

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to open file '%s'", filePath)
	}

	client := getS3Client(c)

	log.Info().Msgf("Uploading file '%s' as ID '%s'", filePath, id)

	S3_BUCKET := c.String("bucket")

	progress := progressbar.DefaultBytes(fileSize)
	reader := io.MultiReader(file, progress)

	startTime := time.Now()
	_, err = client.PutObject(context.Background(), S3_BUCKET, id, reader, fileSize, minio.PutObjectOptions{ContentType: "application/octet-stream", Progress: progress})
	elapsedTime := time.Since(startTime)
	if err != nil {
		log.Err(err).Msg("Failed to upload")
	}

	log.Info().Msgf("Uploaded file with '%s' in %s", util.GetStringFromByteSize(fileSize), elapsedTime)
	uploadSpeed := float64(fileSize) / elapsedTime.Seconds()
	log.Info().Msgf("Average upload speed: %s/s", util.GetStringFromByteSize(int64(uploadSpeed)))
	return nil
}

func remove(c *cli.Context) error {
	if c.Args().Len() != 1 {
		log.Fatal().Msg("Please specify an object to remove")
	}
	id := c.Args().First()
	client := getS3Client(c)

	S3_BUCKET := c.String("bucket")
	err := client.RemoveObject(context.Background(), S3_BUCKET, id, minio.RemoveObjectOptions{})
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to remove object '%s'", id)
	}
	log.Info().Msgf("Removed object '%s'", id)
	return nil
}

func sign(c *cli.Context) error {
	if c.Args().Len() != 1 {
		log.Fatal().Msg("Please specify an object sign the URL for")
	}
	id := c.Args().First()
	client := getS3Client(c)
	S3_BUCKET := c.String("bucket")

	// Set request parameters
	requestParams := make(url.Values)
	// requestParams.Set("response-content-disposition", "attachment; filename=\"filename.pdf\"")
	requestParams.Set("response-content-disposition", "attachment;")

	// Gernerate presigned get object url.
	presignedURL, err := client.PresignedGetObject(context.Background(), S3_BUCKET, id, time.Duration(1)*time.Minute, requestParams)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to generate presigned URL for object '%s'", id)
		return err
	}

	log.Info().Msg("Pre-signed URL for object:")
	log.Info().Msg(presignedURL.String())

	return nil
}

func initLogger(c *cli.Context) error {
	setLogOutput()
	if c.Bool("very-verbose") {
		applyLogLevel("trace")
	} else if c.Bool("verbose") {
		applyLogLevel("debug")
	} else {
		applyLogLevel("info")
	}
	log.Info().Msgf("Logger initialized on level '%s'", zerolog.GlobalLevel().String())
	return nil
}

func setLogOutput() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05.000Z"})
}

func applyLogLevel(logLevel string) {
	switch logLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "err":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func getS3Client(c *cli.Context) *minio.Client {
	S3_ENDPOINT := c.String("endpoint")
	S3_PORT := c.Int("port")
	S3_SSL := !c.Bool("insecure")
	S3_ACCESS_KEY := c.String("access-key")
	S3_SECRET_KEY := c.String("secret-key")

	if S3_ENDPOINT == "" {
		log.Fatal().Msg("Please specify an S3 endpoint")
	}
	if S3_PORT == 0 {
		log.Fatal().Msg("Please specify an S3 port")
	}
	if S3_ACCESS_KEY == "" {
		log.Fatal().Msg("Please specify an S3 access key")
	}
	if S3_SECRET_KEY == "" {
		log.Fatal().Msg("Please specify an S3 secret key")
	}

	log.Info().Msgf("Connecting to S3 host '%s' on port '%d'", S3_ENDPOINT, S3_PORT)

	endpoint := fmt.Sprintf("%s:%d", S3_ENDPOINT, S3_PORT)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(S3_ACCESS_KEY, S3_SECRET_KEY, ""),
		Secure: S3_SSL,
	})
	if err != nil {
		log.Fatal().Err(err)
	}

	return client
}

func performance(c *cli.Context) error {
	vus := c.Int("vus")
	if vus == 0 {
		vus = 1
	}

	duration := c.Int("duration")
	if duration == 0 {
		duration = 30
	}

	stringFileSize := c.String("filesize")
	if stringFileSize == "" {
		stringFileSize = "500KiB"
	}
	byteFileSize, err := util.GetByteSizeFromString(stringFileSize)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse file size: '%s'", stringFileSize)
		return err
	}

	log.Info().Msgf("Starting performance test with %d virtual users for %d seconds and a random file of %s ", vus, duration, util.GetStringFromByteSize(byteFileSize))

	client := getS3Client(c)
	S3_BUCKET := c.String("bucket")
	mutex := sync.Mutex{}
	uploadTimes := make([]float64, 0)
	uploadSpeeds := make([]float64, 0)
	downloadTimes := make([]float64, 0)
	downloadSpeeds := make([]float64, 0)
	deleteTimes := make([]float64, 0)
	iterationDelta := 0

	stop := false

	performanceTest := func() {
		id := uuid.New().String()

		randomFile := make([]byte, byteFileSize)
		rand.Read(randomFile)

		startTime := time.Now()
		_, err := client.PutObject(context.Background(), S3_BUCKET, id, bytes.NewReader(randomFile), byteFileSize, minio.PutObjectOptions{ContentType: "application/octet-stream"})

		if err != nil {
			log.Error().Err(err).Msg("Failed to upload")
			return
		} else {
			elapsedTime := time.Since(startTime)
			mutex.Lock()
			iterationDelta++
			uploadTimes = append(uploadTimes, float64(elapsedTime.Milliseconds()))
			uploadSpeeds = append(uploadSpeeds, float64(byteFileSize)/elapsedTime.Seconds())
			mutex.Unlock()
		}

		startTime = time.Now()
		s3Object, err := client.GetObject(context.Background(), S3_BUCKET, id, minio.GetObjectOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to init object download '%s'", id)
			return
		}

		data, err := io.ReadAll(s3Object)
		log.Trace().Msgf("Downloaded %d bytes", len(data))

		if err != nil {
			log.Error().Err(err).Msgf("Failed to download object '%s'", id)
			return
		} else {
			elapsedTime := time.Since(startTime)
			mutex.Lock()
			downloadTimes = append(downloadTimes, float64(elapsedTime.Milliseconds()))
			downloadSpeeds = append(downloadSpeeds, float64(byteFileSize)/elapsedTime.Seconds())
			mutex.Unlock()
		}

		startTime = time.Now()
		err = client.RemoveObject(context.Background(), S3_BUCKET, id, minio.RemoveObjectOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to remove object '%s'", id)
			return
		} else {
			elapsedTime := time.Since(startTime)
			mutex.Lock()
			deleteTimes = append(deleteTimes, float64(elapsedTime.Milliseconds()))
			mutex.Unlock()
		}

		mutex.Lock()
		iterationDelta++
		mutex.Unlock()
	}

	wg := sync.WaitGroup{}

	worker := func(id int) {
		defer wg.Done()
		wg.Add(1)
		for {
			log.Trace().Msgf("Starting upload for worker %d", id)
			performanceTest()
			log.Trace().Msgf("Finished upload for worker %d", id)
			mutex.Lock()
			if stop {
				break
			}
			mutex.Unlock()
		}
		mutex.Unlock()
	}

	for i := 0; i < vus; i++ {
		go worker(i)
	}

	progress := progressbar.Default(-1)

	for i := 0; i < duration; i++ {
		mutex.Lock()
		progress.Add(iterationDelta)
		iterationDelta = 0
		mutex.Unlock()
		time.Sleep(1 * time.Second)
	}

	progress.Finish()
	log.Info().Msg("Finalizing current worker jobs")
	mutex.Lock()
	stop = true
	mutex.Unlock()

	wg.Wait()

	log.Info().Msg("Performance test finished")

	t := table.NewWriter()
	t.SetTitle(fmt.Sprintf("S3 Performance Times | %d VUs | %d seconds | %s file size", vus, duration, util.GetStringFromByteSize(byteFileSize)))
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Operation", "T min [ms]", "T max [ms]", "P50 [ms]", "P90 [ms]", "P99 [ms]", "Mean [ms]", "Std Dev [ms]"})

	t.AppendRow(table.Row{
		"Upload Time",
		fmt.Sprintf("%.1f", util.GetMinFloat64(uploadTimes)),
		fmt.Sprintf("%.1f", util.GetMaxFloat64(uploadTimes)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(uploadTimes, 50)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(uploadTimes, 90)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(uploadTimes, 99)),
		fmt.Sprintf("%.1f", util.GetMean(uploadTimes)),
		fmt.Sprintf("%.1f", util.GetStdDevFloat64(uploadTimes)),
	})

	t.AppendRow(table.Row{
		"Download",
		fmt.Sprintf("%.1f", util.GetMinFloat64(downloadTimes)),
		fmt.Sprintf("%.1f", util.GetMaxFloat64(downloadTimes)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(downloadTimes, 50)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(downloadTimes, 90)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(downloadTimes, 99)),
		fmt.Sprintf("%.1f", util.GetMean(downloadTimes)),
		fmt.Sprintf("%.1f", util.GetStdDevFloat64(downloadTimes)),
	})

	t.AppendRow(table.Row{
		"Delete",
		fmt.Sprintf("%.1f", util.GetMinFloat64(deleteTimes)),
		fmt.Sprintf("%.1f", util.GetMaxFloat64(deleteTimes)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(deleteTimes, 50)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(deleteTimes, 90)),
		fmt.Sprintf("%.1f", util.GetPercentileFloat64(deleteTimes, 99)),
		fmt.Sprintf("%.1f", util.GetMean(deleteTimes)),
		fmt.Sprintf("%.1f", util.GetStdDevFloat64(deleteTimes)),
	})

	t.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	t.Render()

	t = table.NewWriter()
	t.SetTitle(fmt.Sprintf("S3 Performance Speeds | %d VUs | %d seconds | %s file size", vus, duration, util.GetStringFromByteSize(byteFileSize)))
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Operation", "min [MB/s]", "max [MB/s]", "P50 [MB/s]", "P90 [MB/s]", "P99 [MB/s]", "Mean [MB/s]", "Std Dev [MB/s]"})

	t.AppendRow(table.Row{
		"Upload Speed",
		fmt.Sprintf("%.2f", util.GetMinFloat64(uploadSpeeds)/1000000),
		fmt.Sprintf("%.2f", util.GetMaxFloat64(uploadSpeeds)/1000000),
		fmt.Sprintf("%.2f", util.GetPercentileFloat64(uploadSpeeds, 50)/1000000),
		fmt.Sprintf("%.2f", util.GetPercentileFloat64(uploadSpeeds, 90)/1000000),
		fmt.Sprintf("%.2f", util.GetPercentileFloat64(uploadSpeeds, 99)/1000000),
		fmt.Sprintf("%.2f", util.GetMean(uploadSpeeds)/1000000),
		fmt.Sprintf("%.2f", util.GetStdDevFloat64(uploadSpeeds)/1000000),
	})

	t.AppendRow(table.Row{
		"Download Speed",
		fmt.Sprintf("%.2f", util.GetMinFloat64(downloadSpeeds)/1000000),
		fmt.Sprintf("%.2f", util.GetMaxFloat64(downloadSpeeds)/1000000),
		fmt.Sprintf("%.2f", util.GetPercentileFloat64(downloadSpeeds, 50)/1000000),
		fmt.Sprintf("%.2f", util.GetPercentileFloat64(downloadSpeeds, 90)/1000000),
		fmt.Sprintf("%.2f", util.GetPercentileFloat64(downloadSpeeds, 99)/1000000),
		fmt.Sprintf("%.2f", util.GetMean(downloadSpeeds)/1000000),
		fmt.Sprintf("%.2f", util.GetStdDevFloat64(downloadSpeeds)/1000000),
	})

	t.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	t.Render()

	return nil
}

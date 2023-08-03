package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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

	log.Info().Msgf("File '%s' exists with size '%s'", filePath, getHumanReadableSize(fileSize))

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to open file '%s'", filePath)
	}

	client := getS3Client(c)

	log.Info().Msgf("Uploading file '%s' as ID '%s'", filePath, id)

	S3_BUCKET := c.String("bucket")
	progress := pb.New64(fileSize)
	progress.Start()
	startTime := time.Now()
	_, err = client.PutObject(context.Background(), S3_BUCKET, id, bufio.NewReader(file), fileSize, minio.PutObjectOptions{ContentType: "application/octet-stream", Progress: progress})
	elapsedTime := time.Since(startTime)
	if err != nil {
		log.Err(err).Msg("Failed to upload")
	}
	log.Info().Msgf("Uploaded file with '%s' in %s", getHumanReadableSize(fileSize), elapsedTime)
	uploadSpeed := float64(fileSize) / elapsedTime.Seconds()
	log.Info().Msgf("Average upload speed: %s/s", getHumanReadableSize(int64(uploadSpeed)))
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

func initLogger(c *cli.Context) error {
	setLogOutput()
	if c.Bool("very-verbose") {
		applyLogLevel("trace")
	} else if c.Bool("verbose") {
		applyLogLevel("debug")
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

func getHumanReadableSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MiB", float64(size)/(1024*1024))
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB", float64(size)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TiB", float64(size)/(1024*1024*1024*1024))
	}
}

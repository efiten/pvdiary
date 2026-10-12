package main

import (
	"fmt"
	"log"
	"os"

	"pvdiary"
	"pvdiary/internal/api"
	"pvdiary/internal/httpd"
	"pvdiary/internal/importer"
	"pvdiary/internal/ingest"
	"pvdiary/internal/store"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: pvdiary <serve|import>")
	}
	switch os.Args[1] {
	case "serve":
		serve()
	case "import":
		runImport(os.Args[2:])
	default:
		log.Fatalf("unknown command %q", os.Args[1])
	}
}

func serve() {
	dbPath := env("PVDIARY_DB", "pvdiary.db")
	s, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	site, err := siteFromEnv(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	cfg := api.Config{
		EleCost:     atof(env("PVDIARY_ELE_COST", "0.18")),
		CO2:         atof(env("PVDIARY_CO2", "0.596")),
		IngestToken: os.Getenv("INGEST_TOKEN"),
		Lat:         site.Lat,
		Lon:         site.Lon,
		Strings:     site.Strings,
	}
	if cfg.IngestToken == "" {
		log.Fatal("INGEST_TOKEN is required")
	}
	apiH := api.New(s, ingest.New(s), cfg)
	addr := env("PVDIARY_ADDR", "127.0.0.1:8080")
	log.Printf("pvdiary serving on %s (db=%s)", addr, dbPath)
	log.Fatal(httpd.Serve(addr, httpd.Router(apiH, pvdiary.WebFS)))
}

func runImport(args []string) {
	if len(args) < 2 {
		log.Fatal("usage: pvdiary import <dir> <collectorKey>")
	}
	dir, key := args[0], args[1]
	s, err := store.Open(env("PVDIARY_DB", "pvdiary.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	site, err := siteFromEnv(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	meta := store.Collector{Key: key, Name: key, PVWatt: site.PVWatt(), Lat: site.Lat, Lon: site.Lon}
	rep, err := importer.Run(s, dir, meta)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("imported %d days, %d samples\n", rep.Days, rep.Samples)
	for y, k := range rep.YearKWh {
		fmt.Printf("  %d: %.1f kWh\n", y, k)
	}
}

func atof(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

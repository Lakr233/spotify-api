# Spotify Metadata API

A Go-based API for retrieving Spotify metadata.

This service provides an interface to a curated subset of the comprehensive Spotify metadata backup from **Anna's Archive**. For more context, see their blog post about this effort.

## Data

The database used by this API is a compacted and cleaned version of the original Anna's Archive dataset. The cleaning process is documented in `database/README.md`.

## Running the project

This project is containerized using Docker. To run the service, use the following command:

```bash
docker-compose up
```

The API will be available at `http://localhost:8080`.

## API Reference

The OpenAPI specification for this service can be found in `docs/api-references.yml`.

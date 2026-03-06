package gcs

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
)

// Download baixa o conteúdo de um objeto GCS a partir de um URI no formato gs://bucket/object.
func Download(ctx context.Context, gsURI string) ([]byte, error) {
	bucket, object, err := parseGSURI(gsURI)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs: falha ao criar cliente: %w", err)
	}
	defer client.Close()

	r, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs: falha ao abrir objeto %s: %w", gsURI, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gcs: falha ao ler objeto %s: %w", gsURI, err)
	}

	return data, nil
}

// parseGSURI extrai bucket e object de um URI gs://bucket/object/path.
func parseGSURI(gsURI string) (bucket, object string, err error) {
	if !strings.HasPrefix(gsURI, "gs://") {
		return "", "", fmt.Errorf("gcs: URI inválido %q, esperado gs://bucket/object", gsURI)
	}

	path := strings.TrimPrefix(gsURI, "gs://")
	idx := strings.Index(path, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("gcs: URI inválido %q, faltando path do objeto", gsURI)
	}

	return path[:idx], path[idx+1:], nil
}

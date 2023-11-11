/* Copyright 2023 Take Control - Software & Infrastructure

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package impl providers the main implementation of sync server APIs.
package impl

import (
	"bufio"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/takecontrolsoft/sync/server/config"
	"github.com/takecontrolsoft/sync/server/utils"
)

// Upload file handler for uploading large streamed files.
func UploadHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxUploadFileSize)
		reader, err := r.MultipartReader()
		if utils.OnError(err, w, http.StatusBadRequest) {
			return
		}
		mp, err := reader.NextPart()
		if utils.OnError(err, w, http.StatusInternalServerError) {
			return
		}

		_, params, err := mime.ParseMediaType(mp.Header.Get("Content-Disposition"))
		if utils.OnError(err, w, http.StatusInternalServerError) {
			return
		}
		deviceId := params["name"]
		filename := params["filename"]

		b := bufio.NewReader(mp)
		n, _ := b.Peek(512)
		fileType := http.DetectContentType(n)
		if !utils.ValidateType(fileType, w) {
			utils.RenderError(w, "INVALID_FILE_TYPE", http.StatusBadRequest)
			return
		}

		dirName := filepath.Join(config.UploadDirectory, deviceId)
		err = os.MkdirAll(dirName, os.ModePerm)
		if utils.OnError(err, w, http.StatusInternalServerError) {
			return
		}
		filePath := filepath.Join(dirName, filename)
		f, err := os.Create(filePath)
		if utils.OnError(err, w, http.StatusInternalServerError) {
			return
		}
		defer f.Close()
		var maxSize int64 = config.MaxUploadFileSize
		lmt := io.MultiReader(b, io.LimitReader(mp, maxSize-511))
		written, err := io.Copy(f, lmt)
		if utils.OnError(err, w, http.StatusInternalServerError) {
			return
		}
		if written > maxSize {
			os.Remove(f.Name())
			utils.RenderError(w, "FILE_SIZE_EXCEEDED", http.StatusBadRequest)
			return
		}
	})
}

package main

import (
    "errors"
    "fmt"
    "io"
    "mime/multipart"
    "os"
    "path/filepath"
    "strings"

    "github.com/bodgit/sevenzip"
    "github.com/nwaples/rardecode/v2"
)

const maxArchiveEntries = 8000

// newImportDestination reserves a new directory without overwriting existing mods.
func newImportDestination(root, name string) (string, error) {
    root, err := validDir(root)
    if err != nil { return "", err }
    base := cleanArchiveName(name)
    for i := 1; i < 10000; i++ {
        candidate := base
        if i > 1 { candidate = fmt.Sprintf("%s (%d)", base, i) }
        dest := filepath.Join(root, candidate)
        if err := os.Mkdir(dest, 0755); err == nil {
            return dest, nil
        } else if !os.IsExist(err) { return "", err }
    }
    return "", errors.New("Não foi possível encontrar um nome disponível")
}

type importWriter struct {
    dest string
    total int64
    count int
    hasINI bool
}

// add writes only regular files into the reserved mod folder; any failure
// rolls back the whole import (the caller owns cleanup).
func (w *importWriter) add(name string, size int64, isDir bool, mode os.FileMode, source io.Reader) error {
    rel := strings.TrimSuffix(strings.ReplaceAll(name, "\\", "/"), "/")
    if rel == "" { return nil }
    segments := strings.Split(rel, "/")
    for _, part := range segments {
        if !zipSegmentOK(part) { return fmt.Errorf("Caminho inseguro: %s", name) }
    }
    if mode&os.ModeType != 0 && !isDir { return fmt.Errorf("Link/arquivo especial bloqueado: %s", name) }
    if mode&os.ModeSymlink != 0 { return fmt.Errorf("Link simbólico bloqueado: %s", name) }
    switch strings.ToLower(filepath.Ext(rel)) {
    case ".exe",".msi",".cmd",".bat",".com",".scr",".ps1",".vbs",".vbe",".hta",".lnk",".url",".reg",".pif":
        return fmt.Errorf("Executável ou atalho bloqueado: %s", name)
    }
    if w.count >= maxArchiveEntries { return errors.New("Pacote contém arquivos demais (máximo 8000)") }
    w.count++
    dest := filepath.Join(w.dest, filepath.FromSlash(rel))
    clean, err := filepath.Rel(w.dest, dest)
    if err != nil || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
        return errors.New("Tentativa de escrever fora da pasta do mod")
    }
    if isDir { return os.MkdirAll(dest, 0755) }
    if size < 0 || size > maxExtract-w.total { return errors.New("Conteúdo descompactado ultrapassaria 2 GB") }
    if source == nil { return errors.New("Conteúdo do arquivo indisponível") }
    if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil { return err }
    out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
    if err != nil { return err }
    n, copyErr := io.Copy(out, io.LimitReader(source, size+1))
    closeErr := out.Close()
    if copyErr != nil { return copyErr }
    if closeErr != nil { return closeErr }
    if n != size { return fmt.Errorf("Tamanho incorreto ou pacote corrompido: %s", name) }
    w.total += n
    if strings.EqualFold(filepath.Ext(rel), ".ini") { w.hasINI = true }
    return nil
}

func (w *importWriter) finish() error {
    if !w.hasINI { return errors.New("Nenhum arquivo .ini de mod encontrado") }
    return nil
}

func importArchive(root, name string, body io.Reader) (string, error) {
    switch strings.ToLower(filepath.Ext(name)) {
    case ".zip":
        return importZIP(root, name, body)
    case ".rar", ".7z":
        // Supported by native Go decoders, no external archiver required.
    default:
        return "", errors.New("Formatos aceitos: .zip, .rar e .7z")
    }
    root, err := validDir(root)
    if err != nil { return "", err }
    tmp, err := os.CreateTemp("", "zzz-import-*"+strings.ToLower(filepath.Ext(name)))
    if err != nil { return "", err }
    defer os.Remove(tmp.Name())
    defer tmp.Close()
    n, err := io.Copy(tmp, io.LimitReader(body, maxUpload+1))
    if err != nil { return "", err }
    if n > maxUpload { return "", errors.New("Arquivo maior que 1 GB") }
    if err := tmp.Close(); err != nil { return "", err }

    // Open and validate the format BEFORE creating a destination.
    if strings.EqualFold(filepath.Ext(name), ".rar") {
        rar, err := rardecode.OpenReader(tmp.Name())
        if err != nil { return "", fmt.Errorf("RAR inválido ou não suportado: %w", err) }
        defer rar.Close()
        dest, err := newImportDestination(root, name)
        if err != nil { return "", err }
        ok := false
        defer func() { if !ok { _ = os.RemoveAll(dest) } }()
        w := &importWriter{dest: dest}
        for {
            header, err := rar.Next()
            if err == io.EOF { break }
            if err != nil { return "", fmt.Errorf("Falha ao ler RAR: %w", err) }
            if header.Mode()&os.ModeSymlink != 0 || (header.Mode()&os.ModeType != 0 && !header.IsDir) { return "", errors.New("RAR contém link ou arquivo especial") }
            if header.UnKnownSize { return "", errors.New("RAR contém arquivo com tamanho desconhecido") }
            mode := header.Mode()
            if err := w.add(header.Name, header.UnPackedSize, header.IsDir, mode, rar); err != nil { return "", err }
        }
        if err := w.finish(); err != nil { return "", err }
        ok = true
        return filepath.Base(dest), nil
    }
    seven, err := sevenzip.OpenReader(tmp.Name())
    if err != nil { return "", fmt.Errorf("7z inválido ou não suportado: %w", err) }
    defer seven.Close()
    if len(seven.File) > maxArchiveEntries { return "", errors.New("Pacote contém arquivos demais") }
    dest, err := newImportDestination(root, name)
    if err != nil { return "", err }
    ok := false
    defer func() { if !ok { _ = os.RemoveAll(dest) } }()
    w := &importWriter{dest: dest}
    for _, file := range seven.File {
        info := file.FileInfo()
        if info.Mode()&os.ModeSymlink != 0 || (info.Mode()&os.ModeType != 0 && !info.IsDir()) {
            return "", errors.New("7z contém link ou arquivo especial")
        }
        if info.IsDir() {
            if err := w.add(file.Name, 0, true, info.Mode(), nil); err != nil { return "", err }
            continue
        }
        if file.UncompressedSize > uint64(maxExtract-w.total) { return "", errors.New("7z descompactado ultrapassaria 2 GB") }
        reader, err := file.Open()
        if err != nil { return "", err }
        err = w.add(file.Name, int64(file.UncompressedSize), false, info.Mode(), reader)
        closeErr := reader.Close()
        if err != nil { return "", err }
        if closeErr != nil { return "", closeErr }
    }
    if err := w.finish(); err != nil { return "", err }
    ok = true
    return filepath.Base(dest), nil
}

// importFolder streams a browser-selected directory using alternating "path"/"file"
// multipart entries, without loading entire folders into browser JS memory.
func importFolder(root, folder string, parts *multipart.Reader) (string, error) {
    dest, err := newImportDestination(root, folder)
    if err != nil { return "", err }
    ok := false
    defer func() { if !ok { _ = os.RemoveAll(dest) } }()
    w := &importWriter{dest: dest}
    for {
        part, err := parts.NextPart()
        if err == io.EOF { break }
        if err != nil { return "", err }
        if part.FormName() != "path" { part.Close(); return "", errors.New("Importação da pasta fora de ordem") }
        pathBytes, err := io.ReadAll(io.LimitReader(part, 2049))
        part.Close()
        if err != nil || len(pathBytes) > 2048 { return "", errors.New("Caminho muito longo") }
        file, err := parts.NextPart()
        if err != nil { return "", errors.New("Arquivo ausente após caminho") }
        if file.FormName() != "file" { file.Close(); return "", errors.New("Arquivo de pasta inválido") }
        // Browser File.size travels as an explicit part, so size limits stay
        // enforceable without buffering each file.
        // The form provides size separately as the next field before file.
        // For streaming unknown-size form entries, copy only until cap.
        err = w.addUnknownSize(string(pathBytes), file)
        file.Close()
        if err != nil { return "", err }
    }
    if err := w.finish(); err != nil { return "", err }
    ok = true
    return filepath.Base(dest), nil
}

// Folder uploads do not include a trusted Content-Length on each multipart
// part, so enforce the same limits while streaming directly to disk.
func (w *importWriter) addUnknownSize(name string, src io.Reader) error {
    rel := strings.TrimSuffix(strings.ReplaceAll(name, "\\", "/"), "/")
    if rel == "" { return errors.New("Nome de arquivo vazio") }
    for _, part := range strings.Split(rel, "/") {
        if !zipSegmentOK(part) { return fmt.Errorf("Caminho inseguro: %s", name) }
    }
    switch strings.ToLower(filepath.Ext(rel)) {
    case ".exe",".msi",".cmd",".bat",".com",".scr",".ps1",".vbs",".vbe",".hta",".lnk",".url",".reg",".pif":
        return fmt.Errorf("Executável ou atalho bloqueado: %s", name)
    }
    if w.count >= maxArchiveEntries { return errors.New("Pasta contém arquivos demais (máximo 8000)") }
    w.count++
    dest := filepath.Join(w.dest, filepath.FromSlash(rel))
    clean, err := filepath.Rel(w.dest, dest)
    if err != nil || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
        return errors.New("Caminho fora da pasta de destino")
    }
    if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil { return err }
    out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
    if err != nil { return err }
    n, copyErr := io.Copy(out, io.LimitReader(src, maxExtract-w.total+1))
    closeErr := out.Close()
    if copyErr != nil { return copyErr }
    if closeErr != nil { return closeErr }
    if n > maxExtract-w.total { return errors.New("Pasta ultrapassaria 2 GB") }
    w.total += n
    if strings.EqualFold(filepath.Ext(rel), ".ini") { w.hasINI = true }
    return nil
}

package storkutil

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type FileManager struct{}

func (*FileManager) IsExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, errors.Wrapf(err, "cannot stat the file: %s", path)
	}
}

func (fm *FileManager) RemoveIfExist(path string) error {
	ok, err := fm.IsExist(path)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err = os.Remove(path); err != nil {
		return errors.Wrapf(err, "cannot remove the file: %s", path)
	}

	return nil
}

func (fm *FileManager) createDirectoryTree(path string) error {
	directory := filepath.Dir(path)
	ok, err := fm.IsExist(directory)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	if err := os.MkdirAll(directory, 0o700); err != nil {
		return errors.Wrapf(err, "cannot create a directory tree: %s", directory)
	}
	return nil
}

func (*FileManager) Read(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read the file: %s", path)
	}
	return content, nil
}

func (fm *FileManager) Write(path string, content []byte) error {
	if err := fm.RemoveIfExist(path); err != nil {
		return err
	}

	if err := fm.createDirectoryTree(path); err != nil {
		return err
	}

	err := os.WriteFile(path, content, 0o600)
	if err != nil {
		return errors.Wrapf(err, "cannot write the file: %s", path)
	}
	return nil
}

// func (fm *FileManager) ReadSafe(path string) ([]byte, error) {
// 	validator, ok := fm.validators[path]
// 	if !ok {
// 		return nil, errors.Errorf("missing validator for: %s", path)
// 	}
// 	content, err := fm.Read(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if err = validator(content); err != nil {
// 		return nil, errors.WithMessagef(err, "validation failed for: %s", path)
// 	}
// 	return content, nil
// }

// func (fm *FileManager) WriteSafe(path string, content []byte) error {
// 	validator, ok := fm.validators[path]
// 	if !ok {
// 		return errors.Errorf("missing validator for: %s", path)
// 	}
// 	if err := validator(content); err != nil {
// 		return errors.WithMessagef(err, "validation failed for: %s", path)
// 	}
// 	return fm.Write(path, content)
// }

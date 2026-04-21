package core

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/pkg/sftp"
)

// CreateSftp 创建sftp客户端
func (sclient *SSHClient) CreateSftp() error {
	err := sclient.GenerateClient()
	if err != nil {
		return err
	}
	client, err := sftp.NewClient(sclient.Client)
	if err != nil {
		return err
	}
	sclient.Sftp = client
	return nil
}

// Mkdirs 创建目录
func (sclient *SSHClient) Mkdirs(path string) error {
	if _, err := sclient.Sftp.Stat(path); isSFTPNotExist(err) {
		return sclient.Sftp.MkdirAll(path)
	} else if err != nil {
		return err
	}
	return nil
}

// Mkdir 创建单层目录
func (sclient *SSHClient) Mkdir(path string) error {
	if _, err := sclient.Sftp.Stat(path); err == nil {
		return fmt.Errorf("path already exists")
	} else if !isSFTPNotExist(err) {
		return err
	}
	return sclient.Sftp.Mkdir(path)
}

// Download 下载文件
func (sclient *SSHClient) Download(srcPath string) (*sftp.File, error) {
	return sclient.Sftp.Open(srcPath)
}

// CreateEmptyFile 创建空文件
func (sclient *SSHClient) CreateEmptyFile(dstPath string) error {
	if _, err := sclient.Sftp.Stat(dstPath); err == nil {
		return fmt.Errorf("path already exists")
	} else if !isSFTPNotExist(err) {
		return err
	}
	dstFile, err := sclient.Sftp.OpenFile(dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY)
	if err != nil {
		return err
	}
	return dstFile.Close()
}

// Rename 重命名文件/文件夹。若目标已存在则失败（SFTP Rename 不覆盖）。
func (sclient *SSHClient) Rename(oldPath, newPath string) error {
	if _, err := sclient.Sftp.Lstat(oldPath); err != nil {
		if isSFTPNotExist(err) {
			return fmt.Errorf("source path does not exist")
		}
		return err
	}
	if _, err := sclient.Sftp.Lstat(newPath); err == nil {
		return fmt.Errorf("path already exists")
	} else if !isSFTPNotExist(err) {
		return err
	}
	return sclient.Sftp.Rename(oldPath, newPath)
}

// Upload 上传文件
func (sclient *SSHClient) Upload(file multipart.File, id, dstPath string) error {
	dstFile, err := sclient.Sftp.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	defer func() {
		RemoveWriteCounter(id)
	}()
	wc := NewWriteCounter(id)
	AddWriteCounter(wc)
	_, err = io.Copy(dstFile, io.TeeReader(file, wc))
	if err != nil {
		return err
	}
	return nil
}

func isSFTPNotExist(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "not exist")
}

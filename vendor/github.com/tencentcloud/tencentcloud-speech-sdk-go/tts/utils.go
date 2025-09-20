package tts

import "os"

func WriteFile(filename string, content []byte) error {
	fout, err := os.Create(filename)
	defer fout.Close()
	if err != nil {
		return err
	}

	_, err = fout.Write(content)
	if err != nil {
		return err
	}
	return nil
}

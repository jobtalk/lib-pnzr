package main

/*
typedef struct{
	char* file;
	char* profile;
	char* kmsKeyID;
	char* region;
	char* externalPath;
}setting;

typedef struct {
	char* msg;
	char* error;
}result;
*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/jobtalk/thor/api"
	"github.com/jobtalk/thor/lib"
	. "github.com/jobtalk/thor/lib/setting"
)

var re = regexp.MustCompile(`.*\.json$`)

type deployConfigure struct {
	*Setting
}

type setting struct {
	file         *string
	profile      *string
	kmsKeyID     *string
	region       *string
	externalPath *string
	cred         *credentials.Credentials
}

func newSetting(s *C.setting) *setting {
	if s == nil {
		return nil
	}
	return &setting{
		aws.String(C.GoString(s.file)),
		aws.String(C.GoString(s.profile)),
		aws.String(C.GoString(s.kmsKeyID)),
		aws.String(C.GoString(s.region)),
		aws.String(C.GoString(s.externalPath)),
		nil,
	}
}

//export deploy
func deploy(s *C.setting) *C.result {
	result, err := deployGo(newSetting(s))
	if err != nil {
		return &C.result{C.CString(""), C.CString(err.Error())}
	}
	return &C.result{C.CString(result), C.CString("")}
}

func isEncrypted(data []byte) bool {
	var buffer = map[string]interface{}{}
	if err := json.Unmarshal(data, &buffer); err != nil {
		return false
	}
	elem, ok := buffer["cipher"]
	if !ok {
		return false
	}
	str, ok := elem.(string)
	if !ok {
		return false
	}

	return len(str) != 0
}

func decrypt(bin []byte, s *setting) ([]byte, error) {
	awsConfig := &aws.Config{
		Credentials: s.cred,
		Region:      s.region,
	}
	kms := lib.NewKMSFromBinary(bin)
	if kms == nil {
		return nil, errors.New(fmt.Sprintf("%v format is illegal", string(bin)))
	}
	plainText, err := kms.SetKeyID(*s.kmsKeyID).SetAWSConfig(awsConfig).Decrypt()
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

func deployGo(s *setting) (string, error) {
	var config = &deployConfigure{}

	s.cred = credentials.NewSharedCredentials("", *s.profile)
	awsConfig := &aws.Config{
		Credentials: s.cred,
		Region:      s.region,
	}

	externalList, err := fileList(*s.externalPath)
	if err != nil {
		return "", err
	}

	if externalList != nil {
		c, err := readConf(externalList, s)
		if err != nil {
			return "", err
		}
		config = c
	} else {
		bin, err := ioutil.ReadFile(*s.file)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(bin, config); err != nil {
			return "", err
		}
	}

	result, err := api.Deploy(awsConfig, config.Setting)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(result), nil
}

func readConf(externalPathList []string, s *setting) (*deployConfigure, error) {
	var root = *s.externalPath
	var ret = &deployConfigure{}
	base, err := ioutil.ReadFile(*s.file)
	baseStr := string(base)
	if err != nil {
		return nil, err
	}
	root = strings.TrimSuffix(root, "/")
	for _, externalPath := range externalPathList {
		external, err := ioutil.ReadFile(root + "/" + externalPath)
		if err != nil {
			return nil, err
		}
		if isEncrypted(external) {
			plain, err := decrypt(external, s)
			if err != nil {
				return nil, err
			}
			external = plain
		}
		baseStr, err = lib.Embedde(baseStr, string(external))
		if err != nil {
			return nil, err
		}
	}
	if err := json.Unmarshal([]byte(baseStr), ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func fileList(root string) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	ret := []string{}
	err := filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if re.MatchString(rel) {
				ret = append(ret, rel)
			}

			return nil
		})

	if err != nil {
		return nil, err
	}

	return ret, nil
}

func main() {
}

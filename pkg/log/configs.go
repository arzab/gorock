package log

import "github.com/sirupsen/logrus"

type Configs logrus.JSONFormatter

func (c *Configs) Format(entry *logrus.Entry) ([]byte, error) {
	return (*logrus.JSONFormatter)(c).Format(entry)
}

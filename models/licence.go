package models

type Licence struct {
	IDBase
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Ref         int64  `json:"ref"`
	Avatar      string `json:"avatar"`
	Creator     int64  `json:"creator"`
	Status      int    `json:"status"`
}

func (licence Licence) TableName() string {
	return "licence"
}

func CreateLicense(licence *Licence) error {
	err := DataBase().Create(licence).Error
	if err != nil {
		return err
	}
	return nil
}

func GetLicenseById(id int64) (*Licence, error) {
	var licence Licence
	err := DataBase().Where("id = ?", id).First(&licence).Error
	if err != nil {
		return nil, err
	}
	return &licence, nil
}

func GetLicenseByName(name string) (*Licence, error) {
	var licence Licence
	err := DataBase().Where("name = ?", name).First(&licence).Error
	if err != nil {
		return nil, err
	}
	return &licence, nil
}

func GetLicenseByRef(ref int64) (*Licence, error) {
	var licence Licence
	err := DataBase().Where("ref = ?", ref).First(&licence).Error
	if err != nil {
		return nil, err
	}
	return &licence, nil
}

func GetLicenseByCreator(creator int64) ([]*Licence, error) {
	var licences []*Licence
	err := DataBase().Where("creator = ?", creator).Find(&licences).Error
	if err != nil {
		return nil, err
	}
	return licences, nil
}

func UpdateLicense(licence *Licence) error {
	err := DataBase().Save(licence).Error
	if err != nil {
		return err
	}
	return nil
}

func DeleteLicense(licence *Licence) error {
	err := DataBase().Delete(licence).Error
	if err != nil {
		return err
	}
	return nil
}

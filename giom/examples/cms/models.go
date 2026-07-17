package main

import "gorm.io/gorm"

type Page struct {
	gorm.Model
	Title      string   `json:"title"`
	Slug       string   `json:"slug" gorm:"uniqueIndex"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	CoverImage string   `json:"coverImage"`
	Images     []string `json:"images" gorm:"serializer:json"`
	Published  bool     `json:"published"`
}

type Tag struct {
	gorm.Model
	Name string `json:"name"`
	Slug string `json:"slug" gorm:"uniqueIndex"`
	URL  string `json:"url" gorm:"-"`
}

type Post struct {
	gorm.Model
	Title      string   `json:"title"`
	Slug       string   `json:"slug" gorm:"uniqueIndex"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	RightBody  string   `json:"rightBody"`
	CoverImage string   `json:"coverImage"`
	Images     []string `json:"images" gorm:"serializer:json"`
	Published  bool     `json:"published"`
	Tags       []Tag    `json:"tags" gorm:"many2many:post_tags"`
	URL        string   `json:"url" gorm:"-"`
}

type MenuItem struct {
	gorm.Model
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	PageID   *uint  `json:"pageId"`
	Page     *Page  `json:"page"`
	TagID    *uint  `json:"tagId"`
	Tag      *Tag   `json:"tag"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Visible  bool   `json:"visible"`
}

type SeedData struct {
	Pages []SeedPage     `yaml:"pages"`
	Tags  []SeedTag      `yaml:"tags"`
	Posts []SeedPost     `yaml:"posts"`
	Menu  []SeedMenuItem `yaml:"menu"`
}

type SeedPage struct {
	Title      string   `yaml:"title"`
	Slug       string   `yaml:"slug"`
	Summary    string   `yaml:"summary"`
	Body       string   `yaml:"body"`
	CoverImage string   `yaml:"coverImage"`
	Images     []string `yaml:"images"`
	Published  bool     `yaml:"published"`
}

type SeedTag struct {
	Name string `yaml:"name"`
	Slug string `yaml:"slug"`
}

type SeedPost struct {
	Title      string   `yaml:"title"`
	Slug       string   `yaml:"slug"`
	Summary    string   `yaml:"summary"`
	Body       string   `yaml:"body"`
	RightBody  string   `yaml:"rightBody"`
	CoverImage string   `yaml:"coverImage"`
	Images     []string `yaml:"images"`
	Published  bool     `yaml:"published"`
	Tags       []string `yaml:"tags"`
}

type SeedMenuItem struct {
	Label    string `yaml:"label"`
	Kind     string `yaml:"kind"`
	Page     string `yaml:"page"`
	Tag      string `yaml:"tag"`
	URL      string `yaml:"url"`
	Position int    `yaml:"position"`
	Visible  bool   `yaml:"visible"`
}

type crumb struct {
	Label, URL string
}

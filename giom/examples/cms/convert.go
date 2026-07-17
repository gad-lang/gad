package main

import "github.com/gad-lang/gad"

func pageValue(p Page) gad.Object {
	return gad.MustNewReflectValue(p)
}

func postValue(p Post) gad.Object {
	p.URL = "/posts/" + p.Slug
	return gad.MustNewReflectValue(p)
}

func tagValue(t Tag) gad.Object {
	t.URL = "/tags/" + t.Slug
	return gad.MustNewReflectValue(t)
}

func postsValue(posts []Post) gad.Object {
	for i := range posts {
		posts[i].URL = "/posts/" + posts[i].Slug
	}
	return gad.MustNewReflectValue(posts)
}

func tagsValue(tags []Tag) gad.Object {
	for i := range tags {
		tags[i].URL = "/tags/" + tags[i].Slug
	}
	return gad.MustNewReflectValue(tags)
}

func crumbsValue(crumbs []crumb) gad.Object {
	return gad.MustNewReflectValue(crumbs)
}

func (a *App) menuItems() []MenuItem {
	var items []MenuItem
	a.DB.Preload("Page").Preload("Tag").Where("visible = ?", true).Order("position ASC, id ASC").Find(&items)
	for i := range items {
		url := items[i].URL
		if items[i].Kind == "page" && items[i].Page != nil {
			url = "/pages/" + items[i].Page.Slug
		}
		if items[i].Kind == "tag" && items[i].Tag != nil {
			url = "/tags/" + items[i].Tag.Slug
		}
		items[i].URL = url
	}
	return items
}

func (a *App) latestPosts(limit int) ([]Post, error) {
	var posts []Post
	err := a.DB.Preload("Tags").Where("published = ?", true).Order("created_at DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

func (a *App) postsByTag(tagID uint, page, limit int) ([]Post, int64, error) {
	var total int64
	base := a.DB.Model(&Post{}).Joins("JOIN post_tags ON post_tags.post_id = posts.id").Where("posts.published = ? AND post_tags.tag_id = ?", true, tagID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var posts []Post
	err := base.Preload("Tags").Order("posts.created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&posts).Error
	return posts, total, err
}

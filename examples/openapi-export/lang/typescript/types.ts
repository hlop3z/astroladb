export interface Types {
    BlogComment?: BlogComment;
    BlogPost?:    BlogPost;
    BlogUser?:    BlogUser;
    [property: string]: any;
}

export interface BlogComment {
    author_id:  string;
    body:       string;
    created_at: Date;
    id:         string;
    post_id:    string;
    updated_at: Date;
}

export interface BlogPost {
    author_id:  string;
    content:    string;
    created_at: Date;
    featured:   boolean;
    id:         string;
    slug:       string;
    status:     string;
    title:      string;
    updated_at: Date;
}

export interface BlogUser {
    balance:      number;
    bio?:         string;
    created_at:   Date;
    display_name: string;
    email:        string;
    id:           string;
    level:        number;
    money:        string;
    the_date:     Date;
    the_day:      Date;
    the_file:     string;
    the_object:   { [key: string]: any };
    the_time:     string;
    updated_at:   Date;
    username:     string;
}

s/paramEquals(req, /r.ParamIs(/;
s/paramValue(req, /r.Param(/;
s/bail(w, err)/r.Fail(route.Oops(err, "FIXME need an error message"))/
s/JSON(w, /r.OK(/

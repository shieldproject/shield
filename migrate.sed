s/paramEquals(req, /r.ParamIs(/;
s/paramValue(req, /r.Param(/;
s/bail(w, err)/r.Fail(route.Oops(err, "FIXME need an error message"))/
s/bailWithError(w, err)/r.Fail(route.Oops(err, "FIXME need an error message"))/
s/JSON(w, /r.OK(/
s/w.WriteHeader(404)/r.Fail(route.NotFound(nil, "FIXME need a 404 message"))/

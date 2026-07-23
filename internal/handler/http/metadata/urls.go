package metadata

//func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
//	var userID int
//	var user model.User
//	userToken, err := r.Cookie("Authorization")
//	if errors.Is(err, http.ErrNoCookie) {
//		userID, err = h.userService.GetByIP(*h.ctx, r.RemoteAddr)
//		if err != nil && userID == -1 {
//			user, err = h.userService.Create(*h.ctx, r.RemoteAddr)
//			if err != nil {
//				handler.ErrorResponse(w, "could not create user", http.StatusInternalServerError, h.logger)
//				return -1, err
//			}
//			userID = user.ID
//		}
//		token, servErr := h.userService.Login(userID, h.cfg)
//		if servErr != nil {
//			return -1, fmt.Errorf("could not create token: %w", servErr)
//		}
//		http.SetCookie(w, &http.Cookie{
//			Name:     "Authorization",
//			Value:    token,
//			Expires:  time.Now().Add(h.cfg.TokenExp),
//			HttpOnly: true,
//			Path:     "/",
//		})
//	} else {
//		userID = service.GetUserID(userToken.Value, h.cfg)
//		if userID == -1 {
//			user, err = h.userService.Create(*h.ctx, r.RemoteAddr)
//			if err != nil {
//				return -1, fmt.Errorf("could not create token: %w", err)
//			}
//			token, err := h.userService.Login(user.ID, h.cfg)
//			if err != nil {
//				handler.ErrorResponse(w, err.Error(), http.StatusBadRequest, h.logger)
//				return -1, fmt.Errorf("could not create token: %w", err)
//			}
//			http.SetCookie(w, &http.Cookie{
//				Name:     "Authorization",
//				Value:    token,
//				Expires:  time.Now().Add(h.cfg.TokenExp),
//				HttpOnly: true,
//				Path:     "/",
//			})
//		}
//	}
//	return userID, nil
//}

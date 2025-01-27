package service

import (
	"bufio"
	"bytes"
	"context"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/errors"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"go.uber.org/zap"
	"time"
)

const collectionKey = "key"

func (s *Service) UploadKeysFile(ctx context.Context, req *grpc.PlatformKeysFileRequest, res *grpc.PlatformKeysFileResponse) error {
	scanner := bufio.NewScanner(bytes.NewReader(req.File))
	count, err := s.keyRepository.CountKeysByProductPlatform(req.KeyProductId, req.PlatformId)

	if err != nil {
		zap.S().Errorf(errors.KeyErrorNotFound.Message, "err", err.Error(), "keyProductId", req.KeyProductId, "platformId", req.PlatformId)
		res.Status = pkg.ResponseStatusNotFound
		res.Message = errors.KeyErrorNotFound
		return nil
	}

	res.TotalCount = int32(count)

	// Process key by line
	for scanner.Scan() {
		key := &billing.Key{
			Id:           bson.NewObjectId().Hex(),
			Code:         scanner.Text(),
			KeyProductId: req.KeyProductId,
			PlatformId:   req.PlatformId,
		}

		if err := s.keyRepository.Insert(key); err != nil {
			zap.S().Errorf(errors.KeyErrorFailedToInsert.Message, "err", err, "key", key)
			continue
		}

		res.TotalCount++
		res.KeysProcessed++
	}

	// tell about errors
	if err = scanner.Err(); err != nil {
		zap.S().Errorf(errors.KeyErrorFileProcess.Message, "err", err.Error())
		res.Message = errors.KeyErrorFileProcess
		res.Status = pkg.ResponseStatusBadData
		return nil
	}

	res.Status = pkg.ResponseStatusOk

	return nil
}

func (s *Service) GetAvailableKeysCount(ctx context.Context, req *grpc.GetPlatformKeyCountRequest, res *grpc.GetPlatformKeyCountResponse) error {
	count, err := s.keyRepository.CountKeysByProductPlatform(req.KeyProductId, req.PlatformId)

	if err != nil {
		zap.S().Errorf(errors.KeyErrorNotFound.Message, "err", err.Error(), "keyProductId", req.KeyProductId, "platformId", req.PlatformId)
		res.Status = pkg.ResponseStatusNotFound
		res.Message = errors.KeyErrorNotFound
		return nil
	}

	res.Count = int32(count)
	res.Status = pkg.ResponseStatusOk

	return nil
}

func (s *Service) GetKeyByID(ctx context.Context, req *grpc.KeyForOrderRequest, res *grpc.GetKeyForOrderRequestResponse) error {
	key, err := s.keyRepository.GetById(req.KeyId)

	if err != nil {
		zap.S().Errorf(errors.KeyErrorNotFound.Message, "err", err.Error(), "keyId", req.KeyId)
		res.Status = pkg.ResponseStatusNotFound
		res.Message = errors.KeyErrorNotFound
		return nil
	}

	res.Key = key

	return nil
}

func (s *Service) ReserveKeyForOrder(ctx context.Context, req *grpc.PlatformKeyReserveRequest, res *grpc.PlatformKeyReserveResponse) error {
	zap.S().Infow("[ReserveKeyForOrder] called", "order_id", req.OrderId, "platform_id", req.PlatformId, "KeyProductId", req.KeyProductId)
	key, err := s.keyRepository.ReserveKey(req.KeyProductId, req.PlatformId, req.OrderId, req.Ttl)
	if err != nil {
		zap.S().Errorf(
			errors.KeyErrorReserve.Message,
			"err", err,
			"keyProductId", req.KeyProductId,
			"platformId", req.PlatformId,
			"orderId", req.OrderId,
			"ttl", req.Ttl,
		)
		res.Status = pkg.ResponseStatusBadData
		res.Message = errors.KeyErrorReserve
		return nil
	}

	zap.S().Infow("[ReserveKeyForOrder] reserved key", "req.order_id", req.OrderId, "key.order_id", key.OrderId, "key.id", key.Id, "key.RedeemedAt", key.RedeemedAt, "key.KeyProductId", key.KeyProductId)

	res.KeyId = key.Id
	res.Status = pkg.ResponseStatusOk

	return nil
}

func (s *Service) FinishRedeemKeyForOrder(ctx context.Context, req *grpc.KeyForOrderRequest, res *grpc.GetKeyForOrderRequestResponse) error {
	key, err := s.keyRepository.FinishRedeemById(req.KeyId)

	if err != nil {
		zap.S().Errorf(errors.KeyErrorFinish.Message, "err", err, "keyId", req.KeyId)
		res.Status = pkg.ResponseStatusSystemError
		res.Message = errors.KeyErrorFinish
		return nil
	}

	res.Key = key
	res.Status = pkg.ResponseStatusOk

	return nil
}

func (s *Service) CancelRedeemKeyForOrder(ctx context.Context, req *grpc.KeyForOrderRequest, res *grpc.EmptyResponseWithStatus) error {
	_, err := s.keyRepository.CancelById(req.KeyId)

	if err != nil {
		zap.S().Errorf(errors.KeyErrorCanceled.Message, "err", err, "keyId", req.KeyId)
		res.Status = pkg.ResponseStatusSystemError
		res.Message = errors.KeyErrorCanceled
		return nil
	}

	res.Status = pkg.ResponseStatusOk

	return nil
}

func (s *Service) KeyDaemonProcess() (int, error) {
	counter := 0
	keys, err := s.keyRepository.FindUnfinished()

	if err != nil {
		return counter, err
	}

	for _, key := range keys {
		_, err = s.keyRepository.CancelById(key.Id)

		if err != nil {
			zap.S().Errorf(errors.KeyErrorCanceled.Message, "err", err, "keyId", key.Id)
			continue
		}

		counter++
	}

	return counter, nil
}

type KeyRepositoryInterface interface {
	Insert(*billing.Key) error
	GetById(string) (*billing.Key, error)
	ReserveKey(string, string, string, int32) (*billing.Key, error)
	CancelById(string) (*billing.Key, error)
	FinishRedeemById(string) (*billing.Key, error)
	CountKeysByProductPlatform(string, string) (int, error)
	FindUnfinished() ([]*billing.Key, error)
}

func newKeyRepository(svc *Service) *Key {
	s := &Key{svc: svc}
	return s
}

func (h *Key) Insert(key *billing.Key) error {
	err := h.svc.db.Collection(collectionKey).Insert(key)

	if err != nil {
		return err
	}

	return nil
}

func (h *Key) GetById(id string) (*billing.Key, error) {
	key := &billing.Key{}
	err := h.svc.db.Collection(collectionKey).Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(key)

	if err != nil {
		return nil, err
	}

	return key, nil
}

func (h *Key) ReserveKey(keyProductId string, platformId string, orderId string, ttl int32) (*billing.Key, error) {
	key := &billing.Key{}
	duration := time.Second * time.Duration(ttl)
	query := bson.M{
		"key_product_id": bson.ObjectIdHex(keyProductId),
		"platform_id":    platformId,
		"order_id":       nil,
	}
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"reserved_to": time.Now().UTC().Add(duration),
				"order_id":    bson.ObjectIdHex(orderId),
			},
		},
		ReturnNew: true,
	}

	info, err := h.svc.db.Collection(collectionKey).Find(query).Limit(1).Apply(change, key)

	if err == mgo.ErrNotFound {
		return nil, errors.KeyErrorNotFound
	}

	if err != nil {
		return nil, err
	}

	if info.Updated == 0 {
		return nil, errors.KeyErrorNotFound
	}

	return key, nil
}

func (h *Key) CancelById(id string) (*billing.Key, error) {
	key := &billing.Key{}
	query := bson.M{"_id": bson.ObjectIdHex(id)}
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"reserved_to": "",
				"order_id":    nil,
			},
		},
		ReturnNew: true,
	}

	info, err := h.svc.db.Collection(collectionKey).Find(query).Limit(1).Apply(change, key)

	if err != nil {
		return nil, err
	}

	if info.Updated == 0 {
		return nil, errors.KeyErrorNotFound
	}

	return key, nil
}

func (h *Key) FinishRedeemById(id string) (*billing.Key, error) {
	key := &billing.Key{}
	query := bson.M{"_id": bson.ObjectIdHex(id)}
	change := mgo.Change{
		Update: bson.M{
			"$set": bson.M{
				"reserved_to": "",
				"redeemed_at": time.Now().UTC(),
			},
		},
		ReturnNew: true,
	}

	info, err := h.svc.db.Collection(collectionKey).Find(query).Limit(1).Apply(change, key)

	if err != nil {
		return nil, err
	}

	if info.Updated == 0 {
		return nil, errors.KeyErrorNotFound
	}

	return key, nil
}

func (h *Key) CountKeysByProductPlatform(keyProductId string, platformId string) (int, error) {
	query := bson.M{
		"key_product_id": bson.ObjectIdHex(keyProductId),
		"platform_id":    platformId,
		"order_id":       nil,
	}

	return h.svc.db.Collection(collectionKey).Find(query).Count()
}

func (h *Key) FindUnfinished() ([]*billing.Key, error) {
	var keys []*billing.Key

	query := bson.M{
		"reserved_to": bson.M{
			"$gt": time.Time{},
			"$lt": time.Now().UTC(),
		},
	}

	if err := h.svc.db.Collection(collectionKey).Find(query).All(&keys); err != nil {
		return nil, err
	}

	return keys, nil
}

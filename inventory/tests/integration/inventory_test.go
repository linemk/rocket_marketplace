//go:build integration

package integration

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	grpcmiddleware "github.com/linemk/rocket-shop/platform/pkg/middleware/grpc"
	inventoryV1 "github.com/linemk/rocket-shop/shared/pkg/proto/inventory/v1"
)

const testSessionUUID = "test-session-uuid-12345"

var _ = Describe("InventoryService", func() {
	var (
		ctx             context.Context
		cancel          context.CancelFunc
		inventoryClient inventoryV1.InventoryServiceClient
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(suiteCtx)

		// Добавляем session UUID в контекст для всех gRPC вызовов
		ctx = metadata.AppendToOutgoingContext(ctx, grpcmiddleware.SessionUUIDHeader, testSessionUUID)

		// Чистим коллекцию перед каждым тестом
		err := env.ClearPartsCollection(ctx)
		Expect(err).ToNot(HaveOccurred(), "ожидали успешную очистку коллекции parts перед тестом")

		// Создаём gRPC клиент
		conn, err := grpc.NewClient(
			env.App.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		Expect(err).ToNot(HaveOccurred(), "ожидали успешное подключение к gRPC приложению")

		inventoryClient = inventoryV1.NewInventoryServiceClient(conn)
	})

	AfterEach(func() {
		// Чистим коллекцию после теста
		err := env.ClearPartsCollection(ctx)
		Expect(err).ToNot(HaveOccurred(), "ожидали успешную очистку коллекции parts")

		cancel()
	})

	Describe("GetPart", func() {
		var partUUID string

		BeforeEach(func() {
			// Вставляем тестовую деталь
			var err error
			partUUID, err = env.InsertTestPart(ctx)
			Expect(err).ToNot(HaveOccurred(), "ожидали успешную вставку тестовой детали в MongoDB")
		})

		It("должен успешно возвращать деталь по UUID", func() {
			resp, err := inventoryClient.GetPart(ctx, &inventoryV1.GetPartRequest{
				Uuid: partUUID,
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetPart()).ToNot(BeNil())
			Expect(resp.GetPart().Uuid).To(Equal(partUUID))
			Expect(resp.GetPart().GetName()).ToNot(BeEmpty())
			Expect(resp.GetPart().GetDescription()).ToNot(BeEmpty())
			Expect(resp.GetPart().GetPrice()).To(BeNumerically(">", 0))
			Expect(resp.GetPart().GetDimensions()).ToNot(BeNil())
			Expect(resp.GetPart().GetManufacturer()).ToNot(BeNil())
			Expect(resp.GetPart().GetCreatedAt()).ToNot(BeNil())
		})

		It("должен возвращать ошибку для несуществующего UUID", func() {
			resp, err := inventoryClient.GetPart(ctx, &inventoryV1.GetPartRequest{
				Uuid: "non-existent-uuid",
			})

			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		})
	})

	Describe("ListParts", func() {
		BeforeEach(func() {
			// Вставляем несколько тестовых деталей
			for i := 0; i < 5; i++ {
				_, err := env.InsertTestPart(ctx)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("должен возвращать список всех деталей без фильтра", func() {
			resp, err := inventoryClient.ListParts(ctx, &inventoryV1.ListPartsRequest{
				Filter: &inventoryV1.PartsFilter{},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetParts()).ToNot(BeEmpty())
			Expect(len(resp.GetParts())).To(Equal(5))
		})

		It("должен фильтровать детали по UUID", func() {
			// Вставляем специальную деталь для фильтрации
			testPart := env.GetTestPart()
			partUUID, err := env.InsertTestPartWithData(ctx, testPart)
			Expect(err).ToNot(HaveOccurred())

			resp, err := inventoryClient.ListParts(ctx, &inventoryV1.ListPartsRequest{
				Filter: &inventoryV1.PartsFilter{
					Uuids: []string{partUUID},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetParts()).To(HaveLen(1))
			Expect(resp.GetParts()[0].Uuid).To(Equal(partUUID))
			Expect(resp.GetParts()[0].Name).To(Equal(testPart.Name))
		})

		It("должен фильтровать детали по категории", func() {
			// Вставляем деталь с определенной категорией
			testPart := env.GetTestPart()
			testPart.Category = inventoryV1.Category_CATEGORY_ENGINE
			_, err := env.InsertTestPartWithData(ctx, testPart)
			Expect(err).ToNot(HaveOccurred())

			resp, err := inventoryClient.ListParts(ctx, &inventoryV1.ListPartsRequest{
				Filter: &inventoryV1.PartsFilter{
					Categories: []inventoryV1.Category{inventoryV1.Category_CATEGORY_ENGINE},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.GetParts()).ToNot(BeEmpty())

			// Проверяем, что все детали имеют правильную категорию
			for _, part := range resp.GetParts() {
				Expect(part.Category).To(Equal(inventoryV1.Category_CATEGORY_ENGINE))
			}
		})
	})

	Describe("Полный жизненный цикл", func() {
		It("должен корректно работать с деталями", func() {
			// 1. Вставляем тестовую деталь
			testPart := env.GetTestPart()
			partUUID, err := env.InsertTestPartWithData(ctx, testPart)
			Expect(err).ToNot(HaveOccurred())
			Expect(partUUID).ToNot(BeEmpty())

			// 2. Получаем деталь через API
			getResp, err := inventoryClient.GetPart(ctx, &inventoryV1.GetPartRequest{
				Uuid: partUUID,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(getResp.GetPart()).ToNot(BeNil())
			Expect(getResp.GetPart().Uuid).To(Equal(partUUID))
			Expect(getResp.GetPart().Name).To(Equal(testPart.Name))

			// 3. Получаем список деталей с фильтром по UUID
			listResp, err := inventoryClient.ListParts(ctx, &inventoryV1.ListPartsRequest{
				Filter: &inventoryV1.PartsFilter{
					Uuids: []string{partUUID},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(listResp.GetParts()).To(HaveLen(1))
			Expect(listResp.GetParts()[0].Uuid).To(Equal(partUUID))

			// 4. Проверяем фильтрацию по имени
			listByNameResp, err := inventoryClient.ListParts(ctx, &inventoryV1.ListPartsRequest{
				Filter: &inventoryV1.PartsFilter{
					Names: []string{testPart.Name},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(listByNameResp.GetParts()).ToNot(BeEmpty())

			found := false
			for _, part := range listByNameResp.GetParts() {
				if part.Uuid == partUUID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "деталь должна быть найдена по имени")
		})
	})
})

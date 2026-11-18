import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:price/core/models/account.dart';
import 'package:price/core/services/api_client.dart';
import 'package:price/core/services/api_service.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/core/services/auth_storage.dart';
import 'package:price/core/services/cart_storage.dart';
import 'package:price/core/services/location_service.dart';
import 'package:price/modules/user/user_page.dart';
import 'package:provider/provider.dart';

import 'comparison_test_helpers.dart';

/// Stands in for price-auth, price-user and the reverse-geocoding route of
/// price-back, all behind one MockClient.
class FakeAccountBackend {
  final Map<String, String> passwords = {'ana': 'senha-forte-1'};
  final List<Map<String, dynamic>> lists = [];
  final List<http.Request> requests = [];

  /// Answers 401 to every authenticated call, as an expired token would.
  bool rejectToken = false;

  /// 'ok', 'not-found' or 'down'.
  String reverse = 'ok';
  Map<String, dynamic> place = {
    'street': 'Avenida Miguel Rosa',
    'neighborhood': 'Centro',
    'city': 'Teresina',
    'state': 'PI',
    'cep': '64001-490',
    'label': 'Avenida Miguel Rosa, Centro, Teresina - PI',
  };

  int _nextId = 1;

  http.Response _json(Object body, [int status = 200]) =>
      http.Response(jsonEncode(body), status, headers: {'content-type': 'application/json; charset=utf-8'});

  http.Response _error(String message, int status) => _json({'error': message}, status);

  Map<String, dynamic> _authResponse(String username) => {
        'access_token': 'token-$username',
        'user': {'id': 'id-$username', 'username': username, 'role': 'user'},
      };

  http.Client get client => MockClient((request) async {
        requests.add(request);
        final path = request.url.path;

        if (path == '/auth/signup' || path == '/auth/login') {
          final body = jsonDecode(request.body) as Map<String, dynamic>;
          final username = body['username'] as String;
          final password = body['password'] as String;
          if (path == '/auth/signup') {
            if (passwords.containsKey(username)) return _error('username already taken', 409);
            if (password.length < 8) return _error('password must be at least 8 characters', 400);
            passwords[username] = password;
            return _json(_authResponse(username), 201);
          }
          if (passwords[username] != password) return _error('invalid username or password', 401);
          return _json(_authResponse(username));
        }

        if (path.startsWith('/api/shopping-lists')) {
          final auth = request.headers['Authorization'] ?? '';
          if (rejectToken || !auth.startsWith('Bearer token-')) return _error('invalid or expired token', 401);

          if (request.method == 'GET') return _json(lists);
          if (request.method == 'POST') {
            final body = jsonDecode(request.body) as Map<String, dynamic>;
            final items = body['items'] as List<dynamic>;
            final existing = lists.indexWhere((l) => (l['name'] as String).toLowerCase() == (body['name'] as String).toLowerCase());
            final list = {
              'id': existing >= 0 ? lists[existing]['id'] : 'list-${_nextId++}',
              'name': body['name'],
              'item_count': items.length,
              'items': items,
              'created_at': '2026-09-19T10:00:00Z',
              'updated_at': '2026-09-19T10:00:00Z',
            };
            if (existing >= 0) {
              lists.removeAt(existing);
            }
            lists.insert(0, list);
            return _json(list, existing >= 0 ? 200 : 201);
          }
          if (request.method == 'DELETE') {
            final id = path.split('/').last;
            final before = lists.length;
            lists.removeWhere((l) => l['id'] == id);
            return lists.length == before ? _error('shopping list not found', 404) : http.Response('', 204);
          }
        }

        if (path == '/api/geocode/reverse') {
          switch (reverse) {
            case 'not-found':
              return _error('no address found for these coordinates', 404);
            case 'down':
              return _error('address lookup is unavailable right now', 502);
          }
          return _json(place);
        }

        if (path == '/api/comparisons/list') return _json(comparisonJson());
        return _json([]);
      });

  List<http.Request> where(String method, String path) =>
      requests.where((r) => r.method == method && r.url.path == path).toList();
}

class _Gps extends LocationService {
  UserLocation position;
  int reads = 0;

  _Gps(this.position);

  @override
  Future<UserLocation> currentLocation() async {
    reads++;
    return position;
  }
}

AppState buildAccountState(
  FakeAccountBackend backend, {
  AuthStorage? authStorage,
  CartStorage? cartStorage,
  LocationService? location,
}) {
  final api = ApiClient(baseUrl: 'http://test', client: backend.client);
  return AppState(
    api: ApiService(productsClient: api, authClient: api, userClient: api),
    location: location,
    storage: cartStorage ?? MemoryCartStorage(),
    authStorage: authStorage ?? MemoryAuthStorage(),
  );
}

/// Puts a rice and a coffee on the shopping list.
void fillCart(AppState state) {
  state.updateQuantity(product('p-arroz', catalogId: 'arroz', name: 'Arroz Camil 5kg', price: 22.69), 2);
  state.updateQuantity(product('p-cafe', catalogId: 'cafe', name: 'Café Melitta 500g', price: 25.9), 1);
}

void main() {
  group('account', () {
    test('signing up starts a session, saves it and loads the saved lists', () async {
      final backend = FakeAccountBackend();
      final storage = MemoryAuthStorage();
      final state = buildAccountState(backend, authStorage: storage);

      expect(state.isLoggedIn, isFalse);
      await state.signUp('maria', 'senha-forte-2');

      expect(state.isLoggedIn, isTrue);
      expect(state.user?.username, 'maria');
      expect(storage.saved?.token, 'token-maria');
      await Future<void>.delayed(Duration.zero);
      expect(backend.where('GET', '/api/shopping-lists'), hasLength(1));
      expect(backend.where('GET', '/api/shopping-lists').single.headers['Authorization'], 'Bearer token-maria');
    });

    test('a wrong password says so in Portuguese and does not sign in', () async {
      final state = buildAccountState(FakeAccountBackend());

      await expectLater(
        state.logIn('ana', 'errada'),
        throwsA(isA<ApiException>().having((e) => e.message, 'message', 'Usuário ou senha incorretos.')),
      );
      expect(state.isLoggedIn, isFalse);
    });

    test('a taken username and a short password are explained', () async {
      final state = buildAccountState(FakeAccountBackend());

      await expectLater(
        state.signUp('ana', 'senha-forte-9'),
        throwsA(isA<ApiException>().having((e) => e.message, 'message', 'Esse nome de usuário já está em uso.')),
      );
      await expectLater(
        state.signUp('novo', '123'),
        throwsA(isA<ApiException>().having((e) => e.message, 'message', 'A senha precisa ter pelo menos 8 caracteres.')),
      );
      expect(state.isLoggedIn, isFalse);
    });

    test('the session comes back after restarting the app', () async {
      final backend = FakeAccountBackend();
      final storage = MemoryAuthStorage(
        const AuthSession(token: 'token-ana', user: AppUser(id: 'id-ana', username: 'ana')),
      );
      backend.lists.add({
        'id': 'list-9',
        'name': 'Mês passado',
        'item_count': 1,
        'items': [
          {'product': product('p1', name: 'Feijão').toJson(), 'quantity': 3},
        ],
        'created_at': '2026-09-01T10:00:00Z',
        'updated_at': '2026-09-01T10:00:00Z',
      });

      final state = buildAccountState(backend, authStorage: storage);
      await state.restoreSession();

      expect(state.user?.username, 'ana');
      expect(state.savedLists.map((l) => l.name), ['Mês passado']);
      expect(state.savedLists.single.items.single.quantity, 3);
    });

    test('an expired token signs the person out', () async {
      final backend = FakeAccountBackend()..rejectToken = true;
      final storage = MemoryAuthStorage(
        const AuthSession(token: 'token-ana', user: AppUser(id: 'id-ana', username: 'ana')),
      );
      final state = buildAccountState(backend, authStorage: storage);

      await state.restoreSession();

      expect(state.isLoggedIn, isFalse);
      expect(storage.saved, isNull);
    });

    test('logging out forgets the account but keeps the shopping list in the cart', () async {
      final storage = MemoryAuthStorage();
      final state = buildAccountState(FakeAccountBackend(), authStorage: storage);
      await state.logIn('ana', 'senha-forte-1');
      fillCart(state);

      await state.logOut();

      expect(state.isLoggedIn, isFalse);
      expect(state.savedLists, isEmpty);
      expect(storage.saved, isNull);
      expect(state.cartItems, hasLength(2));
    });
  });

  group('saved lists', () {
    test('saving stores the cart under a name, with product snapshots and quantities', () async {
      final backend = FakeAccountBackend();
      final state = buildAccountState(backend);
      await state.logIn('ana', 'senha-forte-1');
      fillCart(state);

      final saved = await state.saveCurrentList('Compras do mês');

      expect(saved.name, 'Compras do mês');
      expect(state.savedLists.map((l) => l.name), ['Compras do mês']);
      final sent = jsonDecode(backend.where('POST', '/api/shopping-lists').single.body) as Map<String, dynamic>;
      expect(sent['name'], 'Compras do mês');
      final items = sent['items'] as List<dynamic>;
      expect(items, hasLength(2));
      expect((items.first as Map)['quantity'], 2);
      expect(((items.first as Map)['product'] as Map)['catalog_id'], 'arroz');
    });

    test('saving again under the same name replaces the list instead of duplicating it', () async {
      final state = buildAccountState(FakeAccountBackend());
      await state.logIn('ana', 'senha-forte-1');
      fillCart(state);
      await state.saveCurrentList('Semana');

      state.clearCart();
      state.updateQuantity(product('p-leite', catalogId: 'leite', name: 'Leite'), 1);
      await state.saveCurrentList('semana');

      expect(state.savedLists, hasLength(1));
      expect(state.savedLists.single.itemCount, 1);
    });

    test('an empty cart cannot be saved', () async {
      final backend = FakeAccountBackend();
      final state = buildAccountState(backend);
      await state.logIn('ana', 'senha-forte-1');

      await expectLater(state.saveCurrentList('Vazia'), throwsA(isA<ApiException>()));
      expect(backend.where('POST', '/api/shopping-lists'), isEmpty);
    });

    test('using a saved list replaces the cart, keeps it on the device and compares it', () async {
      final backend = FakeAccountBackend();
      final cartStorage = MemoryCartStorage();
      final state = buildAccountState(backend, cartStorage: cartStorage);
      await state.logIn('ana', 'senha-forte-1');
      fillCart(state);
      final saved = await state.saveCurrentList('Semana');

      state.clearCart();
      state.updateQuantity(product('p-sabao', catalogId: 'sabao', name: 'Sabão'), 5);
      final comparisonsBefore = backend.where('POST', '/api/comparisons/list').length;

      state.useSavedList(saved);
      await Future<void>.delayed(Duration.zero);

      expect(state.cartItems.map((i) => i.product.name), ['Arroz Camil 5kg', 'Café Melitta 500g']);
      expect(state.cartItems.first.quantity, 2);
      expect(cartStorage.saved, hasLength(2));
      expect(backend.where('POST', '/api/comparisons/list').length, greaterThan(comparisonsBefore));

      // Editing the cart afterwards must not change the saved list.
      state.updateQuantity(state.cartItems.first.product, 3);
      expect(state.savedLists.single.items.first.quantity, 2);
    });

    test('deleting removes the list from the account', () async {
      final backend = FakeAccountBackend();
      final state = buildAccountState(backend);
      await state.logIn('ana', 'senha-forte-1');
      fillCart(state);
      final saved = await state.saveCurrentList('Semana');

      await state.deleteSavedList(saved);

      expect(state.savedLists, isEmpty);
      expect(backend.lists, isEmpty);
    });

    test('a token rejected while saving signs out and says the session expired', () async {
      final backend = FakeAccountBackend();
      final state = buildAccountState(backend);
      await state.logIn('ana', 'senha-forte-1');
      fillCart(state);
      backend.rejectToken = true;

      await expectLater(
        state.saveCurrentList('Semana'),
        throwsA(isA<ApiException>().having((e) => e.message, 'message', 'Sua sessão expirou. Entre de novo.')),
      );
      expect(state.isLoggedIn, isFalse);
    });
  });

  group('address', () {
    test('updating the location can also look up the street address', () async {
      final backend = FakeAccountBackend();
      final gps = _Gps(const UserLocation(-5.0892, -42.8016));
      final state = buildAccountState(backend, location: gps);

      await state.updateLocation(lookupAddress: true);

      expect(state.userLocation?.lat, -5.0892);
      expect(state.place?.label, 'Avenida Miguel Rosa, Centro, Teresina - PI');
      expect(state.locationUpdatedAt, isNotNull);
      final lookup = backend.where('GET', '/api/geocode/reverse').single;
      expect(lookup.url.queryParameters, {'lat': '-5.0892', 'lng': '-42.8016'});
    });

    test('a failed lookup keeps the new position and just leaves the address out', () async {
      final backend = FakeAccountBackend()..reverse = 'down';
      final state = buildAccountState(backend, location: _Gps(const UserLocation(-5.0892, -42.8016)));

      await state.updateLocation(lookupAddress: true);

      expect(state.userLocation, isNotNull);
      expect(state.place, isNull);
      expect(state.addressLookupFailed, isTrue);
    });

    test('moving again forgets the old address; without lookupAddress nothing is looked up', () async {
      final backend = FakeAccountBackend();
      final gps = _Gps(const UserLocation(-5.0892, -42.8016));
      final state = buildAccountState(backend, location: gps);
      await state.updateLocation(lookupAddress: true);
      expect(state.place, isNotNull);

      gps.position = const UserLocation(-5.1, -42.7);
      await state.updateLocation();

      expect(state.userLocation?.lat, -5.1);
      expect(state.place, isNull);
      expect(backend.where('GET', '/api/geocode/reverse'), hasLength(1));
    });
  });

  group('Perfil tab', () {
    Future<AppState> pumpUserPage(
      WidgetTester tester,
      FakeAccountBackend backend, {
      LocationService? location,
      bool signedIn = false,
      bool withCart = false,
    }) async {
      tester.view.physicalSize = const Size(411, 900);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      final state = buildAccountState(backend, location: location);
      if (signedIn) await state.logIn('ana', 'senha-forte-1');
      if (withCart) fillCart(state);

      await tester.pumpWidget(
        ChangeNotifierProvider.value(value: state, child: const MaterialApp(home: Scaffold(body: UserPage()))),
      );
      await tester.pumpAndSettle();
      return state;
    }

    testWidgets('signed out: offers to log in or create an account, and no saved lists', (tester) async {
      await pumpUserPage(tester, FakeAccountBackend());

      expect(find.text('Entre para guardar suas listas'), findsOneWidget);
      expect(find.widgetWithText(ElevatedButton, 'Entrar'), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, 'Criar conta'), findsOneWidget);
      expect(find.text('Listas salvas'), findsNothing);
      // The location button is there for everyone.
      expect(find.text('Usar minha localização'), findsOneWidget);
    });

    testWidgets('creating an account from the tab signs in and shows the saved lists area', (tester) async {
      final state = await pumpUserPage(tester, FakeAccountBackend());

      await tester.tap(find.widgetWithText(OutlinedButton, 'Criar conta'));
      await tester.pumpAndSettle();
      expect(find.text('Já tem conta? Entrar'), findsOneWidget);

      // Validation runs before anything is sent.
      await tester.tap(find.widgetWithText(ElevatedButton, 'Criar conta'));
      await tester.pumpAndSettle();
      expect(find.text('De 3 a 32 caracteres, sem espaços.'), findsOneWidget);

      await tester.enterText(find.widgetWithText(TextFormField, 'Usuário'), 'joana');
      await tester.enterText(find.widgetWithText(TextFormField, 'Senha'), 'senha-forte-3');
      await tester.tap(find.widgetWithText(ElevatedButton, 'Criar conta'));
      await tester.pumpAndSettle();

      expect(state.user?.username, 'joana');
      expect(find.text('joana'), findsOneWidget);
      expect(find.text('Listas salvas'), findsOneWidget);
      expect(find.text('Você ainda não salvou nenhuma lista.'), findsOneWidget);
    });

    testWidgets('a wrong password shows the error on the login screen and stays there', (tester) async {
      await pumpUserPage(tester, FakeAccountBackend());

      await tester.tap(find.widgetWithText(ElevatedButton, 'Entrar'));
      await tester.pumpAndSettle();
      await tester.enterText(find.widgetWithText(TextFormField, 'Usuário'), 'ana');
      await tester.enterText(find.widgetWithText(TextFormField, 'Senha'), 'errada');
      await tester.tap(find.widgetWithText(ElevatedButton, 'Entrar'));
      await tester.pumpAndSettle();

      expect(find.text('Usuário ou senha incorretos.'), findsOneWidget);
      expect(find.text('Não tem conta? Criar conta'), findsOneWidget);
    });

    testWidgets('saving the current list, then using it later', (tester) async {
      final backend = FakeAccountBackend();
      final state = await pumpUserPage(tester, backend, signedIn: true, withCart: true);

      await tester.tap(find.text('Salvar lista atual (2 itens)'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'Compras do mês');
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Salvar'));
      await tester.pumpAndSettle();

      expect(find.text('Compras do mês'), findsOneWidget);
      expect(find.textContaining('2 itens · salva em'), findsOneWidget);
      expect(backend.lists, hasLength(1));

      // Empty the cart, then bring the saved list back.
      state.clearCart();
      await tester.pumpAndSettle();
      expect(find.text('Salvar lista atual'), findsOneWidget);
      expect(find.text('Monte uma lista na aba Busca para poder salvá-la.'), findsOneWidget);

      await tester.tap(find.text('Usar'));
      await tester.pumpAndSettle();

      expect(state.cartItems, hasLength(2));
      expect(state.currentTabIndex, 1);
    });

    testWidgets('using a list while the cart has items asks before replacing it', (tester) async {
      final backend = FakeAccountBackend();
      final state = await pumpUserPage(tester, backend, signedIn: true, withCart: true);
      await state.saveCurrentList('Semana');
      state.clearCart();
      state.updateQuantity(product('p-leite', catalogId: 'leite', name: 'Leite'), 1);
      await tester.pumpAndSettle();

      await tester.tap(find.text('Usar'));
      await tester.pumpAndSettle();
      expect(find.text('Usar esta lista?'), findsOneWidget);

      await tester.tap(find.text('Cancelar'));
      await tester.pumpAndSettle();
      expect(state.cartItems.single.product.name, 'Leite');

      await tester.tap(find.text('Usar'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Usar lista'));
      await tester.pumpAndSettle();
      expect(state.cartItems, hasLength(2));
    });

    testWidgets('deleting a saved list asks first', (tester) async {
      final backend = FakeAccountBackend();
      final state = await pumpUserPage(tester, backend, signedIn: true, withCart: true);
      await state.saveCurrentList('Semana');
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('Apagar lista'));
      await tester.pumpAndSettle();
      expect(find.text('Apagar lista?'), findsOneWidget);
      await tester.tap(find.text('Apagar'));
      await tester.pumpAndSettle();

      expect(backend.lists, isEmpty);
      expect(find.text('Você ainda não salvou nenhuma lista.'), findsOneWidget);
    });

    testWidgets('the location button reads the GPS again and shows the new address', (tester) async {
      final backend = FakeAccountBackend();
      final gps = _Gps(const UserLocation(-5.0892, -42.8016));
      await pumpUserPage(tester, backend, location: gps);
      expect(find.textContaining('Sem a posição do aparelho'), findsOneWidget);

      await tester.tap(find.text('Usar minha localização'));
      await tester.pumpAndSettle();

      expect(gps.reads, 1);
      expect(find.text('Avenida Miguel Rosa, Centro, Teresina - PI'), findsOneWidget);
      expect(find.text('Atualizar minha localização'), findsOneWidget);

      // The person moved: a second tap picks up the new address.
      backend.place = {...backend.place, 'label': 'Rua Areolino de Abreu, Centro, Teresina - PI'};
      gps.position = const UserLocation(-5.08, -42.79);
      await tester.tap(find.text('Atualizar minha localização'));
      await tester.pumpAndSettle();

      expect(gps.reads, 2);
      expect(find.text('Rua Areolino de Abreu, Centro, Teresina - PI'), findsOneWidget);
      expect(backend.where('GET', '/api/geocode/reverse'), hasLength(2));
    });

    testWidgets('when the address cannot be found the position is still used and the person is told', (tester) async {
      final backend = FakeAccountBackend()..reverse = 'not-found';
      await pumpUserPage(tester, backend, location: _Gps(const UserLocation(0, 0)));

      await tester.tap(find.text('Usar minha localização'));
      await tester.pumpAndSettle();

      expect(find.textContaining('Não foi possível descobrir o endereço'), findsOneWidget);
    });
  });
}

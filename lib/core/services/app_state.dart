import 'dart:async';

import 'package:flutter/material.dart';
import '../models/account.dart';
import '../models/list_comparison.dart';
import '../models/models.dart';
import '../models/product_search.dart';
import 'api_client.dart' show ApiException;
import 'api_service.dart';
import 'auth_storage.dart';
import 'cart_storage.dart';
import 'location_service.dart';
import 'settings_storage.dart';

class AppState extends ChangeNotifier {
  final ApiService _api;
  final LocationService _location;
  final CartStorage _storage;
  final AuthStorage _authStorage;
  final SettingsStorage _settings;

  AppState({
    ApiService? api,
    LocationService? location,
    CartStorage? storage,
    AuthStorage? authStorage,
    SettingsStorage? settings,
  }) : _api = api ?? ApiService(),
       _location = location ?? const LocationService(),
       _storage = storage ?? const SharedPreferencesCartStorage(),
       _authStorage = authStorage ?? const SharedPreferencesAuthStorage(),
       _settings = settings ?? const SharedPreferencesSettingsStorage();

  /// Markets search radius the person can pick, in km. The default matches
  /// the API's own default radius.
  static const double defaultRadiusKm = 5;
  static const double minRadiusKm = 1;
  static const double maxRadiusKm = 50;

  String? _currentAddress;
  // Device position; null until the user grants it (the API then falls back
  // to its default reference point).
  UserLocation? _userLocation;
  // Street address of [_userLocation] (reverse geocoding) and when the position
  // was read. The address is only looked up on demand, so it can be missing.
  PlaceAddress? _place;
  bool _addressLookupFailed = false;
  bool _isLookingUpAddress = false;
  DateTime? _locationUpdatedAt;
  final List<CartItem> _cartItems = [];
  String _selectedCategory = 'Tudo';
  ProductSort _sort = ProductSort.priceAsc;
  SearchFilters _searchFilters = const SearchFilters();
  String _searchQuery = '';
  int _currentTabIndex = 0;

  // Markets (GET /api/markets), within [_radiusKm] of the person.
  double _radiusKm = defaultRadiusKm;
  List<Market> _markets = [];
  bool _isLoadingMarkets = false;
  String? _marketsError;

  // Search results (GET /api/products/search): the pages loaded so far.
  // Filtering, sorting and paging happen on the server.
  List<Product> _searchResults = [];
  int _searchTotal = 0;
  bool _searchHasMore = false;
  bool _isLoadingProducts = false;
  bool _isLoadingMoreProducts = false;
  String? _productsError;
  String? _loadMoreError;
  Timer? _searchDebounce;
  // A new search makes every request still in flight stale.
  int _searchTicket = 0;
  // Categories with their counts under the current search and filters.
  List<CategoryCount> _categories = [];
  int _categoriesTicket = 0;

  // Shopping-list comparison (POST /api/comparisons/list)
  ListComparison? _comparison;
  bool _isLoadingComparison = false;
  String? _comparisonError;
  // Lets another brand fill a gap when a market lacks the one on the list
  // (only for staples like rice and beans; the server decides which).
  bool _allowEquivalents = true;
  // Rapid list edits fire overlapping requests; only the latest may win.
  int _comparisonTicket = 0;

  // Account: null while signed out. The saved lists belong to the account.
  AuthSession? _session;
  List<SavedList> _savedLists = [];
  bool _isLoadingSavedLists = false;
  String? _savedListsError;

  // Economy (GET /api/savings-history, GET /api/economy/summary)
  List<SavingsHistoryItem> _savingsHistory = [];
  EconomySummary? _economySummary;
  bool _isLoadingEconomy = false;
  String? _economyError;

  // Getters
  String? get currentAddress => _currentAddress;
  UserLocation? get userLocation => _userLocation;
  PlaceAddress? get place => _place;
  bool get addressLookupFailed => _addressLookupFailed;
  bool get isLookingUpAddress => _isLookingUpAddress;
  DateTime? get locationUpdatedAt => _locationUpdatedAt;
  List<CartItem> get cartItems => List.unmodifiable(_cartItems);
  String get selectedCategory => _selectedCategory;
  ProductSort get sort => _sort;
  SearchFilters get searchFilters => _searchFilters;
  String get searchQuery => _searchQuery;
  int get currentTabIndex => _currentTabIndex;

  double get radiusKm => _radiusKm;
  List<Market> get markets => _markets;
  bool get isLoadingMarkets => _isLoadingMarkets;
  String? get marketsError => _marketsError;

  List<Product> get searchResults => _searchResults;
  int get searchTotal => _searchTotal;

  /// Pages of the search loaded so far, and how many there are in all
  /// (pages of [ApiService.searchPageSize]).
  int get searchPagesLoaded => (_searchResults.length / ApiService.searchPageSize).ceil();
  int get searchPageCount => (_searchTotal / ApiService.searchPageSize).ceil();
  bool get searchHasMore => _searchHasMore;
  bool get isLoadingProducts => _isLoadingProducts;
  bool get isLoadingMoreProducts => _isLoadingMoreProducts;
  String? get productsError => _productsError;
  String? get loadMoreError => _loadMoreError;
  List<CategoryCount> get categories => _categories;

  ListComparison? get comparison => _comparison;
  bool get allowEquivalents => _allowEquivalents;
  bool get isLoadingComparison => _isLoadingComparison;
  String? get comparisonError => _comparisonError;

  bool get isLoggedIn => _session != null;
  AppUser? get user => _session?.user;
  List<SavedList> get savedLists => List.unmodifiable(_savedLists);
  bool get isLoadingSavedLists => _isLoadingSavedLists;
  String? get savedListsError => _savedListsError;

  List<SavingsHistoryItem> get savingsHistory => _savingsHistory;
  EconomySummary? get economySummary => _economySummary;
  bool get isLoadingEconomy => _isLoadingEconomy;
  String? get economyError => _economyError;

  int get totalItemsCount {
    int total = 0;
    for (var item in _cartItems) {
      total += item.quantity;
    }
    return total;
  }

  double get defaultTotalPrice {
    double total = 0;
    for (var item in _cartItems) {
      total += item.totalPrice;
    }
    return total;
  }

  // Setters & Actions
  void setTabIndex(int index) {
    _currentTabIndex = index;
    notifyListeners();
  }

  void setAddress(String? address) {
    _currentAddress = address;
    notifyListeners();
  }

  void setSelectedCategory(String category) {
    if (_selectedCategory == category) return;
    _selectedCategory = category;
    notifyListeners();
    refreshSearchResults(withCategories: false);
  }

  void setSort(ProductSort sort) {
    if (_sort == sort) return;
    _sort = sort;
    notifyListeners();
    refreshSearchResults(withCategories: false);
  }

  void setSearchFilters(SearchFilters filters) {
    _searchFilters = filters;
    notifyListeners();
    refreshSearchResults();
  }

  void setSearchQuery(String query) {
    _searchQuery = query;
    notifyListeners();
    _searchDebounce?.cancel();
    _searchDebounce = Timer(const Duration(milliseconds: 350), refreshSearchResults);
  }

  void updateQuantity(Product product, int delta) {
    final index = _cartItems.indexWhere((item) => item.product.id == product.id);
    if (index >= 0) {
      final item = _cartItems[index];
      item.quantity += delta;
      if (item.quantity <= 0) {
        _cartItems.removeAt(index);
      }
    } else if (delta > 0) {
      _cartItems.add(CartItem(product: product, quantity: delta));
    }
    notifyListeners();
    _persistCart();
    refreshComparison();
  }

  void clearCart() {
    _cartItems.clear();
    notifyListeners();
    _persistCart();
    refreshComparison();
  }

  void setAllowEquivalents(bool value) {
    if (_allowEquivalents == value) return;
    _allowEquivalents = value;
    notifyListeners();
    refreshComparison();
  }

  /// Puts back the list saved on the device by a previous session.
  Future<void> restoreCart() async {
    final saved = await _storage.load();
    // If something was added while this was loading, that list wins.
    if (saved.isEmpty || _cartItems.isNotEmpty) return;
    _cartItems.addAll(saved);
    notifyListeners();
    refreshComparison();
  }

  /// Puts back the search radius chosen in a previous session.
  Future<void> restoreRadius() async {
    final saved = await _settings.loadRadiusKm();
    if (saved == null) return;
    _radiusKm = saved.clamp(minRadiusKm, maxRadiusKm).toDouble();
    notifyListeners();
  }

  /// Changes how far the app looks for markets, and reloads everything that
  /// depends on it: the markets list and the shopping-list comparison.
  Future<void> setRadiusKm(double km) async {
    final radius = km.clamp(minRadiusKm, maxRadiusKm).roundToDouble();
    if (radius == _radiusKm) return;
    _radiusKm = radius;
    notifyListeners();
    unawaited(_settings.saveRadiusKm(radius));
    await Future.wait([loadMarkets(), refreshComparison(), if (_searchFilters.nearbyOnly) refreshSearchResults()]);
  }

  void _persistCart() {
    unawaited(_storage.save(List.of(_cartItems)));
  }

  bool isProductInCart(Product product) {
    return _cartItems.any((item) => item.product.id == product.id);
  }

  int getProductQuantity(Product product) {
    final index = _cartItems.indexWhere((item) => item.product.id == product.id);
    return index >= 0 ? _cartItems[index].quantity : 0;
  }

  // --- Markets --------------------------------------------------------

  /// Reads the device GPS (asking for permission if needed) and remembers it
  /// for the markets distance. Throws [LocationException] on failure so the
  /// caller can tell the user why.
  Future<void> detectLocation() async {
    _userLocation = await _location.currentLocation();
    // The old address described the old position.
    _place = null;
    _addressLookupFailed = false;
    _locationUpdatedAt = DateTime.now();
    notifyListeners();
  }

  Future<void> openLocationSettings(LocationIssue issue) => _location.openSettings(issue);

  bool _isUpdatingLocation = false;
  bool get isUpdatingLocation => _isUpdatingLocation;

  /// Reads the GPS again and refreshes everything that depends on where the
  /// user is: the markets' distances and the shopping-list comparison. Throws
  /// [LocationException] when the position can't be read (the previous one is
  /// kept, so the app keeps working). With [lookupAddress] it also finds the
  /// street address of the new position; that lookup never throws, it just
  /// leaves [place] empty and sets [addressLookupFailed].
  Future<void> updateLocation({bool lookupAddress = false}) async {
    if (_isUpdatingLocation) return;
    _isUpdatingLocation = true;
    notifyListeners();
    try {
      await detectLocation();
      _currentAddress = 'Minha localização';
      notifyListeners();
      await Future.wait([
        loadMarkets(),
        refreshComparison(),
        if (_searchFilters.nearbyOnly) refreshSearchResults(),
        if (lookupAddress) _lookUpAddress(),
      ]);
    } finally {
      _isUpdatingLocation = false;
      notifyListeners();
    }
  }

  Future<void> _lookUpAddress() async {
    final location = _userLocation;
    if (location == null) return;
    _isLookingUpAddress = true;
    _addressLookupFailed = false;
    notifyListeners();
    try {
      final place = await _api.reverseGeocode(location.lat, location.lng);
      // The person may have moved again while this was running.
      if (!identical(location, _userLocation)) return;
      _place = place;
    } catch (e) {
      if (!identical(location, _userLocation)) return;
      _place = null;
      _addressLookupFailed = true;
    } finally {
      _isLookingUpAddress = false;
      notifyListeners();
    }
  }

  /// Distance in km from the user to [marketId], or null when it isn't known
  /// (markets not loaded, outside the search radius, or no registered
  /// location for the market).
  double? distanceKmFor(String marketId) {
    for (final market in _markets) {
      if (market.id == marketId) return market.distance;
    }
    return null;
  }

  Future<void> loadMarkets({double? lat, double? lng, double? radiusKm}) async {
    _isLoadingMarkets = true;
    _marketsError = null;
    notifyListeners();
    try {
      _markets = await _api.fetchMarkets(
        lat: lat ?? _userLocation?.lat,
        lng: lng ?? _userLocation?.lng,
        radiusKm: radiusKm ?? _radiusKm,
      );
    } catch (e) {
      _marketsError = e.toString();
    } finally {
      _isLoadingMarkets = false;
      notifyListeners();
    }
  }

  // --- Products / Search -----------------------------------------------

  /// Starts the search over from the first page with the current text,
  /// category, sort and filters. With [withCategories] it also reloads the
  /// category counts (they depend on the text and filters, not on the
  /// category or the order).
  Future<void> refreshSearchResults({bool withCategories = true}) async {
    _searchDebounce?.cancel();
    final ticket = ++_searchTicket;
    _isLoadingProducts = true;
    _isLoadingMoreProducts = false;
    _productsError = null;
    _loadMoreError = null;
    notifyListeners();

    if (withCategories) unawaited(_loadCategories(++_categoriesTicket));
    try {
      final page = await _searchPage(offset: 0);
      if (ticket != _searchTicket) return;
      _searchResults = page.items;
      _searchTotal = page.total;
      _searchHasMore = page.hasMore;
    } catch (e) {
      if (ticket != _searchTicket) return;
      _productsError = e.toString();
      _searchResults = [];
      _searchTotal = 0;
      _searchHasMore = false;
    } finally {
      if (ticket == _searchTicket) {
        _isLoadingProducts = false;
        notifyListeners();
      }
    }
  }

  /// Appends the next page; called as the list nears its end. Does nothing
  /// while another page is loading or when there is no more.
  Future<void> loadMoreSearchResults() async {
    if (!_searchHasMore || _isLoadingProducts || _isLoadingMoreProducts) return;
    final ticket = _searchTicket;
    _isLoadingMoreProducts = true;
    _loadMoreError = null;
    notifyListeners();
    try {
      final page = await _searchPage(offset: _searchResults.length);
      if (ticket != _searchTicket) return;
      final known = {for (final p in _searchResults) p.id};
      _searchResults = [..._searchResults, ...page.items.where((p) => known.add(p.id))];
      _searchTotal = page.total;
      _searchHasMore = page.hasMore && page.items.isNotEmpty;
    } catch (e) {
      if (ticket != _searchTicket) return;
      _loadMoreError = e.toString();
    } finally {
      if (ticket == _searchTicket) {
        _isLoadingMoreProducts = false;
        notifyListeners();
      }
    }
  }

  Future<ProductPage> _searchPage({required int offset}) {
    return _api.searchProducts(
      search: _searchQuery,
      category: _selectedCategory,
      sort: _sort,
      filters: _searchFilters,
      lat: _userLocation?.lat,
      lng: _userLocation?.lng,
      radiusKm: _radiusKm,
      offset: offset,
    );
  }

  Future<void> _loadCategories(int ticket) async {
    try {
      final categories = await _api.fetchCategories(
        search: _searchQuery,
        filters: _searchFilters,
        lat: _userLocation?.lat,
        lng: _userLocation?.lng,
        radiusKm: _radiusKm,
      );
      if (ticket != _categoriesTicket) return;
      _categories = categories;
      notifyListeners();
    } catch (_) {
      // The chips keep the previous counts; the search itself reports errors.
    }
  }

  // --- Comparison --------------------------------------------------------

  Future<void> refreshComparison() async {
    final ticket = ++_comparisonTicket;

    if (_cartItems.isEmpty) {
      _comparison = null;
      _comparisonError = null;
      _isLoadingComparison = false;
      notifyListeners();
      return;
    }

    _isLoadingComparison = true;
    _comparisonError = null;
    notifyListeners();
    try {
      final result = await _api.compareList(
        List.of(_cartItems),
        allowEquivalents: _allowEquivalents,
        lat: _userLocation?.lat,
        lng: _userLocation?.lng,
        // The radius needs a position; without one every market is compared.
        radiusKm: _userLocation == null ? null : _radiusKm,
      );
      if (ticket != _comparisonTicket) return; // superseded by a newer request
      _comparison = result;
    } catch (e) {
      if (ticket != _comparisonTicket) return;
      _comparisonError = e.toString();
    } finally {
      if (ticket == _comparisonTicket) {
        _isLoadingComparison = false;
        notifyListeners();
      }
    }
  }

  // --- Account ---------------------------------------------------------------

  /// Puts back the account signed in by a previous session.
  Future<void> restoreSession() async {
    final saved = await _authStorage.load();
    // Signing in while this was loading wins.
    if (saved == null || _session != null) return;
    _session = saved;
    notifyListeners();
    await loadSavedLists();
  }

  /// Creates an account and signs in. Throws [ApiException] with a message
  /// safe to show (name taken, weak password...).
  Future<void> signUp(String username, String password) async {
    await _startSession(await _api.signUp(username, password));
  }

  /// Signs in. Throws [ApiException] with a message safe to show.
  Future<void> logIn(String username, String password) async {
    await _startSession(await _api.logIn(username, password));
  }

  Future<void> _startSession(AuthSession session) async {
    _session = session;
    _savedLists = [];
    _savedListsError = null;
    notifyListeners();
    await _authStorage.save(session);
    unawaited(loadSavedLists());
  }

  /// Signs out. The shopping list in the cart stays on the device; the saved
  /// lists stay on the account.
  Future<void> logOut() async {
    _session = null;
    _savedLists = [];
    _savedListsError = null;
    _isLoadingSavedLists = false;
    notifyListeners();
    await _authStorage.clear();
  }

  static const _sessionExpiredMessage = 'Sua sessão expirou. Entre de novo.';

  /// Runs an account call; a rejected token means the session is over, so sign
  /// out and say so instead of leaving the person with a screen that can't work.
  Future<T> _asAccount<T>(Future<T> Function(String token) call) async {
    final token = _session?.token;
    if (token == null) throw ApiException('Entre na sua conta para continuar.');
    try {
      return await call(token);
    } on ApiException catch (e) {
      if (!e.isUnauthorized) rethrow;
      // Another call may already have signed out (or in again) meanwhile.
      if (_session?.token == token) await logOut();
      throw ApiException(_sessionExpiredMessage, statusCode: e.statusCode);
    }
  }

  Future<void> loadSavedLists() async {
    if (_session == null) return;
    _isLoadingSavedLists = true;
    _savedListsError = null;
    notifyListeners();
    try {
      final lists = await _asAccount(_api.fetchSavedLists);
      _savedLists = lists;
    } catch (e) {
      // Signed out meanwhile (expired token): nothing left to show an error on.
      if (_session != null) _savedListsError = e.toString();
    } finally {
      _isLoadingSavedLists = false;
      notifyListeners();
    }
  }

  /// Saves the shopping list as it is now under [name] (replacing the saved
  /// list with that name, if any). Throws [ApiException] on failure.
  Future<SavedList> saveCurrentList(String name) async {
    if (_cartItems.isEmpty) throw ApiException('Sua lista está vazia. Adicione itens antes de salvar.');
    final items = List.of(_cartItems);
    final saved = await _asAccount((token) => _api.saveList(token, name, items));
    _savedLists = [saved, ..._savedLists.where((l) => l.id != saved.id)];
    notifyListeners();
    return saved;
  }

  /// Replaces the shopping list with a saved one.
  void useSavedList(SavedList list) {
    _cartItems
      ..clear()
      ..addAll(list.items.map((i) => CartItem(product: i.product, quantity: i.quantity)));
    notifyListeners();
    _persistCart();
    refreshComparison();
  }

  Future<void> deleteSavedList(SavedList list) async {
    await _asAccount((token) => _api.deleteSavedList(token, list.id));
    _savedLists = _savedLists.where((l) => l.id != list.id).toList();
    notifyListeners();
  }

  // --- Economy -------------------------------------------------------------

  /// The savings belong to the account, so this needs someone signed in.
  Future<void> loadEconomyData() async {
    _isLoadingEconomy = true;
    _economyError = null;
    notifyListeners();
    try {
      final history = await _asAccount(_api.fetchSavingsHistory);
      final summary = await _asAccount(_api.fetchEconomySummary);
      _savingsHistory = history;
      _economySummary = summary;
    } catch (e) {
      _economyError = e.toString();
    } finally {
      _isLoadingEconomy = false;
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _searchDebounce?.cancel();
    super.dispose();
  }
}

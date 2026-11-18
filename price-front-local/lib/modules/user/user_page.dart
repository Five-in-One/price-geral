import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/components/location_feedback.dart';
import '../../core/components/radius_selector.dart';
import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/theme/app_theme.dart';
import '../../core/models/account.dart';
import '../../core/services/app_state.dart';
import '../../core/services/location_service.dart';
import 'auth_page.dart';

const _blue = AppColors.brand;
const _ink = AppColors.ink;
const _grey = AppColors.grayLight;

String _two(int n) => n.toString().padLeft(2, '0');

/// "Atualizada às 14:05".
String _timeLabel(DateTime t) => '${_two(t.hour)}:${_two(t.minute)}';

/// "19/09".
String _dateLabel(DateTime t) => '${_two(t.day)}/${_two(t.month)}';

/// The "Perfil" tab: the account (or a way in), where the person is, and the
/// shopping lists they saved.
class UserPage extends StatelessWidget {
  const UserPage({super.key});

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();

    return Scaffold(
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 10),
          children: [
            const HeroBanner(
              eyebrow: 'Perfil',
              title: 'Sua',
              highlight: 'conta',
              emoji: '👤',
            ),
            const SizedBox(height: 16),
            if (appState.isLoggedIn) const _AccountCard() else const _SignedOutCard(),
            // The saved lists come right after the account: they are what it is for.
            if (appState.isLoggedIn) ...[
              const SizedBox(height: 22),
              const _SavedListsSection(),
            ],
            const SizedBox(height: 16),
            const _LocationCard(),
            const SizedBox(height: 16),
            const AppCard(child: RadiusSelector()),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}

BoxDecoration _cardDecoration() => BoxDecoration(
      color: Colors.white,
      borderRadius: BorderRadius.circular(22),
      border: Border.all(color: AppColors.border),
      boxShadow: AppColors.softShadow,
    );

// --- Signed out ---------------------------------------------------------------

class _SignedOutCard extends StatelessWidget {
  const _SignedOutCard();

  void _open(BuildContext context, {required bool signUp}) {
    Navigator.of(context).push(MaterialPageRoute(builder: (_) => AuthPage(startInSignUp: signUp)));
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: _cardDecoration(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Row(
            children: [
              IconTile(Icons.person_outline, size: 46),
              SizedBox(width: 14),
              Expanded(
                child: Text(
                  'Entre para guardar suas listas',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold, color: _ink),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          const Text(
            'Com uma conta você salva as listas de compras que já montou e usa de novo quando quiser.',
            style: TextStyle(fontSize: 13, height: 1.45, color: AppColors.gray),
          ),
          const SizedBox(height: 18),
          ElevatedButton(
            onPressed: () => _open(context, signUp: false),
            child: const Text('Entrar'),
          ),
          const SizedBox(height: 10),
          OutlinedButton(
            onPressed: () => _open(context, signUp: true),
            child: const Text('Criar conta'),
          ),
        ],
      ),
    );
  }
}

// --- Signed in ----------------------------------------------------------------

class _AccountCard extends StatelessWidget {
  const _AccountCard();

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final username = appState.user?.username ?? '';

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: _cardDecoration(),
      child: Row(
        children: [
          CircleAvatar(
            radius: 24,
            backgroundColor: _blue,
            child: Text(
              username.isEmpty ? '?' : username[0].toUpperCase(),
              style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 18),
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  username,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold, color: _ink),
                ),
                const Text('Conta Price', style: TextStyle(fontSize: 12, color: _grey)),
              ],
            ),
          ),
          TextButton.icon(
            onPressed: () => context.read<AppState>().logOut(),
            icon: const Icon(Icons.logout, size: 18),
            label: const Text('Sair'),
            style: TextButton.styleFrom(foregroundColor: AppColors.red),
          ),
        ],
      ),
    );
  }
}

// --- Location -----------------------------------------------------------------

class _LocationCard extends StatelessWidget {
  const _LocationCard();

  Future<void> _update(BuildContext context) async {
    final appState = context.read<AppState>();
    final messenger = ScaffoldMessenger.of(context);
    try {
      await appState.updateLocation(lookupAddress: true);
      messenger.showSnackBar(
        const SnackBar(content: Text('Localização atualizada.'), duration: Duration(seconds: 2)),
      );
    } on LocationException catch (e) {
      showLocationError(messenger, appState, e);
    }
  }

  /// What to show for "where am I": the street address when known, else what
  /// the app is going on.
  String _description(AppState appState) {
    final place = appState.place;
    if (place != null && place.label.isNotEmpty) return place.label;
    if (appState.isUpdatingLocation || appState.isLookingUpAddress) return 'Buscando onde você está...';
    if (appState.userLocation != null) {
      return appState.addressLookupFailed
          ? 'Usando a posição do aparelho. Não foi possível descobrir o endereço agora.'
          : 'Usando a posição do aparelho.';
    }
    return 'Sem a posição do aparelho: mostrando os mercados de ${appState.currentAddress ?? 'Teresina'}.';
  }

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final updatedAt = appState.locationUpdatedAt;
    final busy = appState.isUpdatingLocation;
    final hasLocation = appState.userLocation != null;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: _cardDecoration(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Row(
            children: [
              IconTile(Icons.my_location, size: 38),
              SizedBox(width: 12),
              Text('Onde você está', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w800, color: _ink)),
            ],
          ),
          const SizedBox(height: 12),
          Text(
            _description(appState),
            style: const TextStyle(fontSize: 14, height: 1.35, color: _ink, fontWeight: FontWeight.w500),
          ),
          if (updatedAt != null) ...[
            const SizedBox(height: 4),
            Text(
              'Atualizada às ${_timeLabel(updatedAt)}. Os mercados e a comparação usam essa posição.',
              style: const TextStyle(fontSize: 12, color: _grey),
            ),
          ],
          const SizedBox(height: 14),
          OutlinedButton.icon(
            onPressed: busy ? null : () => _update(context),
            icon: busy
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2, color: _blue),
                  )
                : const Icon(Icons.gps_fixed, size: 18),
            label: Text(hasLocation ? 'Atualizar minha localização' : 'Usar minha localização'),
          ),
        ],
      ),
    );
  }
}

// --- Saved lists --------------------------------------------------------------

class _SavedListsSection extends StatelessWidget {
  const _SavedListsSection();

  Future<void> _save(BuildContext context) async {
    final appState = context.read<AppState>();
    final messenger = ScaffoldMessenger.of(context);

    final name = await showDialog<String>(
      context: context,
      builder: (_) => _SaveListDialog(existingNames: appState.savedLists.map((l) => l.name).toList()),
    );
    if (name == null) return;

    try {
      final saved = await appState.saveCurrentList(name);
      messenger.showSnackBar(SnackBar(content: Text('Lista "${saved.name}" salva.')));
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _use(BuildContext context, SavedList list) async {
    final appState = context.read<AppState>();
    final messenger = ScaffoldMessenger.of(context);

    if (appState.cartItems.isNotEmpty) {
      final replace = await _confirm(
        context,
        title: 'Usar esta lista?',
        message: 'Sua lista atual será trocada por "${list.name}". Se quiser guardá-la, salve antes.',
        action: 'Usar lista',
      );
      if (replace != true) return;
    }

    appState.useSavedList(list);
    appState.setTabIndex(1); // the "Lista" tab
    messenger.showSnackBar(SnackBar(content: Text('Lista "${list.name}" carregada.')));
  }

  Future<void> _delete(BuildContext context, SavedList list) async {
    final appState = context.read<AppState>();
    final messenger = ScaffoldMessenger.of(context);

    final confirmed = await _confirm(
      context,
      title: 'Apagar lista?',
      message: '"${list.name}" será removida da sua conta.',
      action: 'Apagar',
      destructive: true,
    );
    if (confirmed != true) return;

    try {
      await appState.deleteSavedList(list);
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final lists = appState.savedLists;
    final cartCount = appState.cartItems.length;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const Row(
          children: [
            IconTile(Icons.bookmarks_outlined, size: 38),
            SizedBox(width: 12),
            Text('Listas salvas', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900, color: _ink)),
          ],
        ),
        const SizedBox(height: 14),
        ElevatedButton.icon(
          onPressed: cartCount == 0 ? null : () => _save(context),
          icon: const Icon(Icons.bookmark_add_outlined, size: 18),
          label: Text(cartCount == 0
              ? 'Salvar lista atual'
              : 'Salvar lista atual ($cartCount ${cartCount == 1 ? 'item' : 'itens'})'),
        ),
        if (cartCount == 0)
          const Padding(
            padding: EdgeInsets.only(top: 6),
            child: Text(
              'Monte uma lista na aba Busca para poder salvá-la.',
              style: TextStyle(fontSize: 12, color: _grey),
            ),
          ),
        const SizedBox(height: 16),
        if (appState.isLoadingSavedLists && lists.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 24),
            child: Center(child: CircularProgressIndicator(color: _blue)),
          )
        else if (appState.savedListsError != null && lists.isEmpty)
          ErrorRetry(message: appState.savedListsError!, onRetry: () => appState.loadSavedLists())
        else if (lists.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 16),
            child: Text(
              'Você ainda não salvou nenhuma lista.',
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 13, color: _grey),
            ),
          )
        else
          for (final list in lists)
            _SavedListTile(
              list: list,
              onUse: () => _use(context, list),
              onDelete: () => _delete(context, list),
            ),
      ],
    );
  }
}

class _SavedListTile extends StatelessWidget {
  final SavedList list;
  final VoidCallback onUse;
  final VoidCallback onDelete;

  const _SavedListTile({required this.list, required this.onUse, required this.onDelete});

  @override
  Widget build(BuildContext context) {
    final items = list.itemCount;
    final subtitle = '$items ${items == 1 ? 'item' : 'itens'} · salva em ${_dateLabel(list.updatedAt)}';

    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Container(
        padding: const EdgeInsets.fromLTRB(14, 12, 6, 12),
        decoration: _cardDecoration(),
        child: Row(
          children: [
            const IconTile(Icons.receipt_long_outlined, size: 40),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    list.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold, color: _ink),
                  ),
                  const SizedBox(height: 2),
                  Text(subtitle, style: const TextStyle(fontSize: 12, color: _grey)),
                ],
              ),
            ),
            TextButton(
              onPressed: onUse,
              style: TextButton.styleFrom(foregroundColor: _blue),
              child: const Text('Usar', style: TextStyle(fontWeight: FontWeight.bold)),
            ),
            IconButton(
              tooltip: 'Apagar lista',
              onPressed: onDelete,
              icon: const Icon(Icons.delete_outline, size: 20, color: _grey),
            ),
          ],
        ),
      ),
    );
  }
}

Future<bool?> _confirm(
  BuildContext context, {
  required String title,
  required String message,
  required String action,
  bool destructive = false,
}) {
  return showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(title),
      content: Text(message),
      actions: [
        TextButton(onPressed: () => Navigator.of(ctx).pop(false), child: const Text('Cancelar')),
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(true),
          style: destructive ? TextButton.styleFrom(foregroundColor: const Color(0xFFB3261E)) : null,
          child: Text(action),
        ),
      ],
    ),
  );
}

/// Asks for the name to save the current list under, and warns when that name
/// belongs to a list that would be replaced.
class _SaveListDialog extends StatefulWidget {
  final List<String> existingNames;

  const _SaveListDialog({required this.existingNames});

  @override
  State<_SaveListDialog> createState() => _SaveListDialogState();
}

class _SaveListDialogState extends State<_SaveListDialog> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  String get _name => _controller.text.trim();

  bool get _replacesExisting {
    final lower = _name.toLowerCase();
    return widget.existingNames.any((n) => n.toLowerCase() == lower);
  }

  void _submit() {
    if (_name.isNotEmpty) Navigator.of(context).pop(_name);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Salvar lista'),
      content: TextField(
        controller: _controller,
        autofocus: true,
        maxLength: 60,
        textInputAction: TextInputAction.done,
        textCapitalization: TextCapitalization.sentences,
        onChanged: (_) => setState(() {}),
        onSubmitted: (_) => _submit(),
        decoration: InputDecoration(
          labelText: 'Nome da lista',
          hintText: 'Ex.: Compras do mês',
          helperText: _replacesExisting ? 'Já existe uma lista com esse nome: ela será substituída.' : null,
          helperMaxLines: 2,
        ),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Cancelar')),
        TextButton(onPressed: _name.isEmpty ? null : _submit, child: const Text('Salvar')),
      ],
    );
  }
}

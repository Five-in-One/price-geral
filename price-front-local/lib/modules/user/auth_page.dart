import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/services/app_state.dart';
import '../../core/theme/app_theme.dart';

/// Log in or create an account. Closes itself once the person is signed in.
class AuthPage extends StatefulWidget {
  /// Opens on "Criar conta" instead of "Entrar".
  final bool startInSignUp;

  const AuthPage({super.key, this.startInSignUp = false});

  @override
  State<AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends State<AuthPage> {
  static const _blue = AppColors.brand;

  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();

  late bool _isSignUp = widget.startInSignUp;
  bool _obscurePassword = true;
  bool _isSubmitting = false;
  String? _error;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _toggleMode() {
    setState(() {
      _isSignUp = !_isSignUp;
      _error = null;
    });
    _formKey.currentState?.reset();
  }

  String? _validateUsername(String? value) {
    final username = (value ?? '').trim();
    if (username.length < 3 || username.length > 32 || username.contains(RegExp(r'\s'))) {
      return 'De 3 a 32 caracteres, sem espaços.';
    }
    return null;
  }

  String? _validatePassword(String? value) {
    final password = value ?? '';
    if (password.isEmpty) return 'Digite a senha.';
    if (_isSignUp && password.length < 8) return 'Pelo menos 8 caracteres.';
    return null;
  }

  Future<void> _submit() async {
    if (_isSubmitting || !(_formKey.currentState?.validate() ?? false)) return;

    final appState = context.read<AppState>();
    final navigator = Navigator.of(context);
    final username = _usernameController.text.trim();
    final password = _passwordController.text;

    setState(() {
      _isSubmitting = true;
      _error = null;
    });
    try {
      if (_isSignUp) {
        await appState.signUp(username, password);
      } else {
        await appState.logIn(username, password);
      }
      if (!mounted) return;
      navigator.pop();
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        scrolledUnderElevation: 0,
        title: const PriceLogoHeader(iconSize: 18, fontSize: 16),
        centerTitle: true,
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
          child: Form(
            key: _formKey,
            child: AppCard(
              padding: const EdgeInsets.fromLTRB(22, 24, 22, 20),
              radius: 28,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Align(
                    alignment: Alignment.centerLeft,
                    child: EyebrowPill(_isSignUp ? 'Nova conta' : 'Bem-vindo de volta'),
                  ),
                  const SizedBox(height: 14),
                  Text(
                    _isSignUp ? 'Criar conta' : 'Entrar',
                    style: const TextStyle(
                      fontSize: 30,
                      fontWeight: FontWeight.w900,
                      color: AppColors.ink,
                      letterSpacing: -0.8,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    _isSignUp
                        ? 'Com uma conta você guarda suas listas de compras e usa de novo quando quiser.'
                        : 'Entre para ver as listas de compras que você salvou.',
                    style: const TextStyle(fontSize: 14, height: 1.45, color: AppColors.gray),
                  ),
                  const SizedBox(height: 26),
                  TextFormField(
                    controller: _usernameController,
                    enabled: !_isSubmitting,
                    textInputAction: TextInputAction.next,
                    autocorrect: false,
                    enableSuggestions: false,
                    autofillHints: const [AutofillHints.username],
                    decoration: const InputDecoration(labelText: 'Usuário', prefixIcon: Icon(Icons.person_outline)),
                    validator: _validateUsername,
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _passwordController,
                    enabled: !_isSubmitting,
                    obscureText: _obscurePassword,
                    textInputAction: TextInputAction.done,
                    onFieldSubmitted: (_) => _submit(),
                    autofillHints: [_isSignUp ? AutofillHints.newPassword : AutofillHints.password],
                    decoration: InputDecoration(
                      labelText: 'Senha',
                      helperText: _isSignUp ? 'Pelo menos 8 caracteres.' : null,
                      prefixIcon: const Icon(Icons.lock_outline),
                      suffixIcon: IconButton(
                        tooltip: _obscurePassword ? 'Mostrar senha' : 'Ocultar senha',
                        icon: Icon(_obscurePassword ? Icons.visibility_outlined : Icons.visibility_off_outlined),
                        onPressed: () => setState(() => _obscurePassword = !_obscurePassword),
                      ),
                    ),
                    validator: _validatePassword,
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: 16),
                    Container(
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(color: AppColors.redBg, borderRadius: BorderRadius.circular(16)),
                      child: Text(
                        _error!,
                        style: const TextStyle(fontSize: 13, color: Color(0xFFB3261E), fontWeight: FontWeight.w500),
                      ),
                    ),
                  ],
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: _isSubmitting ? null : _submit,
                    style: ElevatedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 18)),
                    child: _isSubmitting
                        ? const SizedBox(
                            height: 20,
                            width: 20,
                            child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2),
                          )
                        : Text(
                            _isSignUp ? 'Criar conta' : 'Entrar',
                            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                          ),
                  ),
                  const SizedBox(height: 12),
                  TextButton(
                    onPressed: _isSubmitting ? null : _toggleMode,
                    child: Text(
                      _isSignUp ? 'Já tem conta? Entrar' : 'Não tem conta? Criar conta',
                      style: const TextStyle(fontWeight: FontWeight.w700, color: _blue),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

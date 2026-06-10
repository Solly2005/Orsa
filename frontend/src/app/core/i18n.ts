// Runtime translation dictionary for the ORSA UI.
// Keys are dotted namespaces; each maps to a per-language string.
// "ORSA" is a brand name and is intentionally left untranslated.
// Missing translations fall back to English, then to the raw key.

export type LangCode = 'en' | 'ar' | 'es' | 'fr' | 'de' | 'hi' | 'zh';

type Entry = Partial<Record<LangCode, string>>;

export const TRANSLATIONS: Record<string, Entry> = {
  // ── Navigation ─────────────────────────────────────────────
  'nav.chat': { en: 'Chat', ar: 'المحادثة', es: 'Chat', fr: 'Discussion', de: 'Chat', hi: 'चैट', zh: '对话' },
  'nav.profile': { en: 'Profile', ar: 'الملف الشخصي', es: 'Perfil', fr: 'Profil', de: 'Profil', hi: 'प्रोफ़ाइल', zh: '个人资料' },
  'nav.settings': { en: 'Settings', ar: 'الإعدادات', es: 'Ajustes', fr: 'Paramètres', de: 'Einstellungen', hi: 'सेटिंग्स', zh: '设置' },
  'nav.consent': { en: 'Consent', ar: 'الموافقة', es: 'Consentimiento', fr: 'Consentement', de: 'Einwilligung', hi: 'सहमति', zh: '同意' },
  'nav.signOut': { en: 'Sign out', ar: 'تسجيل الخروج', es: 'Cerrar sesión', fr: 'Se déconnecter', de: 'Abmelden', hi: 'साइन आउट', zh: '退出登录' },
  'nav.login': { en: 'Log in', ar: 'تسجيل الدخول', es: 'Iniciar sesión', fr: 'Connexion', de: 'Anmelden', hi: 'लॉग इन', zh: '登录' },
  'nav.try': { en: 'Try ORSA', ar: 'جرّب ORSA', es: 'Prueba ORSA', fr: 'Essayer ORSA', de: 'ORSA testen', hi: 'ORSA आज़माएँ', zh: '试用 ORSA' },
  'nav.theme.light': { en: 'Light', ar: 'فاتح', es: 'Claro', fr: 'Clair', de: 'Hell', hi: 'लाइट', zh: '浅色' },
  'nav.theme.system': { en: 'System', ar: 'النظام', es: 'Sistema', fr: 'Système', de: 'System', hi: 'सिस्टम', zh: '系统' },
  'nav.theme.dark': { en: 'Dark', ar: 'داكن', es: 'Oscuro', fr: 'Sombre', de: 'Dunkel', hi: 'डार्क', zh: '深色' },

  // ── Common ─────────────────────────────────────────────────
  'common.email': { en: 'Email', ar: 'البريد الإلكتروني', es: 'Correo electrónico', fr: 'E-mail', de: 'E-Mail', hi: 'ईमेल', zh: '电子邮件' },
  'common.password': { en: 'Password', ar: 'كلمة المرور', es: 'Contraseña', fr: 'Mot de passe', de: 'Passwort', hi: 'पासवर्ड', zh: '密码' },

  // ── Landing ────────────────────────────────────────────────
  'landing.eyebrow': { en: 'Your AI health companion', ar: 'رفيقك الصحي بالذكاء الاصطناعي', es: 'Tu compañero de salud con IA', fr: 'Votre compagnon santé par IA', de: 'Ihr KI-Gesundheitsbegleiter', hi: 'आपका एआई स्वास्थ्य साथी', zh: '您的 AI 健康助手' },
  'landing.meet': { en: 'Meet', ar: 'تعرّف على', es: 'Conoce a', fr: 'Découvrez', de: 'Das ist', hi: 'मिलिए', zh: '认识' },
  'landing.lead': {
    en: 'Calm triage support, threaded conversations, health profile controls, and document intake in one responsive workspace.',
    ar: 'دعم فرز هادئ، ومحادثات متسلسلة، وضوابط للملف الصحي، واستلام للمستندات في مساحة عمل واحدة متجاوبة.',
    es: 'Apoyo de triaje tranquilo, conversaciones por hilos, controles del perfil de salud y recepción de documentos en un único espacio de trabajo adaptable.',
    fr: "Un soutien au triage serein, des conversations par fils, des contrôles du profil de santé et la réception de documents dans un seul espace de travail réactif.",
    de: 'Ruhige Triage-Unterstützung, verkettete Gespräche, Steuerung des Gesundheitsprofils und Dokumentenannahme in einem reaktionsfähigen Arbeitsbereich.',
    hi: 'एक ही उत्तरदायी कार्यक्षेत्र में शांत ट्राइएज सहायता, थ्रेडेड बातचीत, स्वास्थ्य प्रोफ़ाइल नियंत्रण और दस्तावेज़ ग्रहण।',
    zh: '在一个自适应的工作区中提供平稳的分诊支持、线程化对话、健康档案控制和文档接收。'
  },
  'landing.openChat': { en: 'Open chat', ar: 'افتح المحادثة', es: 'Abrir chat', fr: 'Ouvrir la discussion', de: 'Chat öffnen', hi: 'चैट खोलें', zh: '打开对话' },
  'landing.tryOrsa': { en: 'Try ORSA', ar: 'جرّب ORSA', es: 'Prueba ORSA', fr: 'Essayer ORSA', de: 'ORSA testen', hi: 'ORSA आज़माएँ', zh: '试用 ORSA' },
  'landing.createAccount': { en: 'Create account', ar: 'إنشاء حساب', es: 'Crear cuenta', fr: 'Créer un compte', de: 'Konto erstellen', hi: 'खाता बनाएँ', zh: '创建账户' },
  'landing.trust.consent': { en: 'Consent first', ar: 'الموافقة أولاً', es: 'Consentimiento primero', fr: "Consentement d'abord", de: 'Einwilligung zuerst', hi: 'पहले सहमति', zh: '同意优先' },
  'landing.trust.escalate': { en: 'Escalate-only triage', ar: 'فرز تصعيدي فقط', es: 'Triaje solo de escalada', fr: 'Triage à escalade uniquement', de: 'Nur eskalierende Triage', hi: 'केवल-एस्केलेट ट्राइएज', zh: '仅升级式分诊' },
  'landing.trust.audit': { en: 'Audit-ready', ar: 'جاهز للتدقيق', es: 'Listo para auditoría', fr: "Prêt pour l'audit", de: 'Audit-bereit', hi: 'ऑडिट के लिए तैयार', zh: '可供审计' },
  'landing.preview.brand': { en: 'ORSA Your Health Companion', ar: 'ORSA رفيقك الصحي', es: 'ORSA Tu compañero de salud', fr: 'ORSA Votre compagnon santé', de: 'ORSA Ihr Gesundheitsbegleiter', hi: 'ORSA आपका स्वास्थ्य साथी', zh: 'ORSA 您的健康助手' },
  'landing.preview.assistant': {
    en: 'Tell me what symptoms you are experiencing and when they started.',
    ar: 'أخبرني ما الأعراض التي تعاني منها ومتى بدأت.',
    es: 'Dime qué síntomas tienes y cuándo comenzaron.',
    fr: 'Dites-moi quels symptômes vous ressentez et quand ils ont commencé.',
    de: 'Sagen Sie mir, welche Symptome Sie haben und wann sie begonnen haben.',
    hi: 'मुझे बताएँ कि आपको कौन से लक्षण हैं और वे कब शुरू हुए।',
    zh: '请告诉我您有哪些症状以及它们何时开始。'
  },
  'landing.preview.user': {
    en: 'Chest tightness and shortness of breath for 30 minutes.',
    ar: 'ضيق في الصدر وصعوبة في التنفس لمدة 30 دقيقة.',
    es: 'Opresión en el pecho y dificultad para respirar durante 30 minutos.',
    fr: "Oppression thoracique et essoufflement depuis 30 minutes.",
    de: 'Engegefühl in der Brust und Atemnot seit 30 Minuten.',
    hi: '30 मिनट से छाती में जकड़न और सांस लेने में तकलीफ़।',
    zh: '胸闷和呼吸急促已持续 30 分钟。'
  },
  'landing.preview.alert': {
    en: 'Urgent symptoms detected. Escalation is never silently reduced.',
    ar: 'تم رصد أعراض عاجلة. لا يتم تخفيض التصعيد بصمت أبداً.',
    es: 'Síntomas urgentes detectados. La escalada nunca se reduce en silencio.',
    fr: "Symptômes urgents détectés. L'escalade n'est jamais réduite en silence.",
    de: 'Dringende Symptome erkannt. Eine Eskalation wird niemals stillschweigend reduziert.',
    hi: 'तत्काल लक्षण पाए गए। एस्केलेशन को कभी चुपचाप कम नहीं किया जाता।',
    zh: '检测到紧急症状。升级绝不会被悄悄降级。'
  },
  'landing.preview.uploads': { en: '3 / 5 uploads used today', ar: '٣ / ٥ تحميلات مُستخدمة اليوم', es: '3 / 5 cargas usadas hoy', fr: "3 / 5 téléversements utilisés aujourd'hui", de: '3 / 5 Uploads heute genutzt', hi: 'आज 3 / 5 अपलोड उपयोग किए गए', zh: '今日已用 3 / 5 次上传' },
  'landing.preview.ready': { en: 'PDF + image ready', ar: 'PDF + صورة جاهزة', es: 'PDF + imagen listos', fr: 'PDF + image prêts', de: 'PDF + Bild bereit', hi: 'PDF + छवि तैयार', zh: 'PDF + 图像就绪' },
  'landing.feat1.title': { en: 'Threaded care conversations', ar: 'محادثات رعاية متسلسلة', es: 'Conversaciones de atención por hilos', fr: 'Conversations de soins par fils', de: 'Verkettete Pflegegespräche', hi: 'थ्रेडेड देखभाल बातचीत', zh: '线程化护理对话' },
  'landing.feat1.desc': {
    en: 'Search, restore, and continue conversations with a compact, familiar chat pattern.',
    ar: 'ابحث واستعد وتابع المحادثات بنمط محادثة مألوف ومُدمج.',
    es: 'Busca, restaura y continúa conversaciones con un patrón de chat compacto y familiar.',
    fr: 'Recherchez, restaurez et poursuivez les conversations avec un modèle de discussion compact et familier.',
    de: 'Suchen, wiederherstellen und Gespräche mit einem kompakten, vertrauten Chat-Muster fortsetzen.',
    hi: 'एक संक्षिप्त, परिचित चैट पैटर्न के साथ बातचीत खोजें, पुनर्स्थापित करें और जारी रखें।',
    zh: '以紧凑、熟悉的聊天模式搜索、恢复和继续对话。'
  },
  'landing.feat2.title': { en: 'Document-aware intake', ar: 'استلام مدرك للمستندات', es: 'Recepción con reconocimiento de documentos', fr: 'Réception consciente des documents', de: 'Dokumentenbewusste Annahme', hi: 'दस्तावेज़-जागरूक ग्रहण', zh: '文档感知接收' },
  'landing.feat2.desc': {
    en: 'Uploads show validation, progress, daily usage, and uncertainty when text or images are not readable.',
    ar: 'تعرض التحميلات التحقق والتقدم والاستخدام اليومي وعدم اليقين عندما يتعذّر قراءة النص أو الصور.',
    es: 'Las cargas muestran validación, progreso, uso diario e incertidumbre cuando el texto o las imágenes no son legibles.',
    fr: "Les téléversements affichent la validation, la progression, l'utilisation quotidienne et l'incertitude lorsque le texte ou les images sont illisibles.",
    de: 'Uploads zeigen Validierung, Fortschritt, tägliche Nutzung und Unsicherheit, wenn Text oder Bilder nicht lesbar sind.',
    hi: 'अपलोड सत्यापन, प्रगति, दैनिक उपयोग और अनिश्चितता दिखाते हैं जब टेक्स्ट या छवियाँ पठनीय नहीं होतीं।',
    zh: '当文本或图像无法读取时，上传会显示验证、进度、每日用量和不确定性。'
  },
  'landing.feat3.title': { en: 'User-controlled memory', ar: 'ذاكرة يتحكّم بها المستخدم', es: 'Memoria controlada por el usuario', fr: "Mémoire contrôlée par l'utilisateur", de: 'Benutzergesteuerter Speicher', hi: 'उपयोगकर्ता-नियंत्रित मेमोरी', zh: '用户可控的记忆' },
  'landing.feat3.desc': {
    en: 'Persona extraction is opt-in, audited, and stored outside the triage workflow.',
    ar: 'استخراج السمات اختياري ومُدقّق ويُخزّن خارج سير عمل الفرز.',
    es: 'La extracción de perfil es opcional, auditada y se almacena fuera del flujo de triaje.',
    fr: "L'extraction de profil est facultative, auditée et stockée en dehors du flux de triage.",
    de: 'Die Profilerfassung ist freiwillig, geprüft und wird außerhalb des Triage-Workflows gespeichert.',
    hi: 'पर्सोना निष्कर्षण वैकल्पिक, ऑडिट किया हुआ और ट्राइएज वर्कफ़्लो के बाहर संग्रहीत है।',
    zh: '人物画像提取为选择性加入、经过审计，并存储在分诊流程之外。'
  },

  // ── Auth (sign in) ─────────────────────────────────────────
  'auth.eyebrow': { en: 'Welcome back', ar: 'مرحباً بعودتك', es: 'Bienvenido de nuevo', fr: 'Bon retour', de: 'Willkommen zurück', hi: 'वापसी पर स्वागत है', zh: '欢迎回来' },
  'auth.title': { en: 'Sign in to ORSA', ar: 'تسجيل الدخول إلى ORSA', es: 'Inicia sesión en ORSA', fr: 'Se connecter à ORSA', de: 'Bei ORSA anmelden', hi: 'ORSA में साइन इन करें', zh: '登录 ORSA' },
  'auth.sub': {
    en: 'Chat and your health profile are available once you are signed in.',
    ar: 'تتوفر المحادثة وملفك الصحي بمجرد تسجيل الدخول.',
    es: 'El chat y tu perfil de salud están disponibles una vez que inicias sesión.',
    fr: 'La discussion et votre profil de santé sont disponibles une fois connecté.',
    de: 'Chat und Ihr Gesundheitsprofil sind nach der Anmeldung verfügbar.',
    hi: 'साइन इन करते ही चैट और आपकी स्वास्थ्य प्रोफ़ाइल उपलब्ध हो जाती है।',
    zh: '登录后即可使用对话和您的健康档案。'
  },
  'auth.signin': { en: 'Sign in', ar: 'تسجيل الدخول', es: 'Iniciar sesión', fr: 'Se connecter', de: 'Anmelden', hi: 'साइन इन', zh: '登录' },
  'auth.signingin': { en: 'Signing in…', ar: 'جارٍ تسجيل الدخول…', es: 'Iniciando sesión…', fr: 'Connexion…', de: 'Anmeldung…', hi: 'साइन इन हो रहा है…', zh: '正在登录…' },
  'auth.or': { en: 'or', ar: 'أو', es: 'o', fr: 'ou', de: 'oder', hi: 'या', zh: '或' },
  'auth.google': { en: 'Continue with Google', ar: 'المتابعة باستخدام Google', es: 'Continuar con Google', fr: 'Continuer avec Google', de: 'Mit Google fortfahren', hi: 'Google के साथ जारी रखें', zh: '使用 Google 继续' },
  'auth.newToOrsa': { en: 'New to ORSA?', ar: 'جديد على ORSA؟', es: '¿Nuevo en ORSA?', fr: 'Nouveau sur ORSA ?', de: 'Neu bei ORSA?', hi: 'ORSA पर नए हैं?', zh: '初次使用 ORSA？' },
  'auth.createAccountLink': { en: 'Create an account', ar: 'أنشئ حساباً', es: 'Crea una cuenta', fr: 'Créer un compte', de: 'Konto erstellen', hi: 'खाता बनाएँ', zh: '创建账户' },
  'auth.whyTitle': { en: 'Why an account?', ar: 'لماذا الحساب؟', es: '¿Por qué una cuenta?', fr: 'Pourquoi un compte ?', de: 'Warum ein Konto?', hi: 'खाता क्यों?', zh: '为什么需要账户？' },
  'auth.why1': { en: 'Chat is available to signed-in members only', ar: 'المحادثة متاحة للأعضاء المسجّلين فقط', es: 'El chat está disponible solo para miembros que han iniciado sesión', fr: 'La discussion est réservée aux membres connectés', de: 'Der Chat ist nur für angemeldete Mitglieder verfügbar', hi: 'चैट केवल साइन-इन सदस्यों के लिए उपलब्ध है', zh: '对话仅向已登录的成员开放' },
  'auth.why2': { en: 'Legal acceptance is captured before account creation', ar: 'يتم تسجيل الموافقة القانونية قبل إنشاء الحساب', es: 'La aceptación legal se registra antes de crear la cuenta', fr: "L'acceptation juridique est enregistrée avant la création du compte", de: 'Die rechtliche Zustimmung wird vor der Kontoerstellung erfasst', hi: 'खाता बनाने से पहले कानूनी स्वीकृति दर्ज की जाती है', zh: '在创建账户前记录法律同意' },
  'auth.why3': { en: 'Location captured with geolocation then IP fallback', ar: 'يُلتقط الموقع عبر تحديد الموقع الجغرافي ثم عنوان IP كبديل', es: 'La ubicación se obtiene por geolocalización y luego por IP', fr: "La localisation est obtenue par géolocalisation puis par IP", de: 'Standort wird per Geolokalisierung und ersatzweise per IP erfasst', hi: 'स्थान जियोलोकेशन से, फिर IP फ़ॉलबैक से लिया जाता है', zh: '通过地理定位获取位置，并以 IP 作为后备' },
  'auth.why4': { en: 'Memory extraction is opt-in and defaults off', ar: 'استخراج الذاكرة اختياري ومُعطّل افتراضياً', es: 'La extracción de memoria es opcional y está desactivada por defecto', fr: "L'extraction de mémoire est facultative et désactivée par défaut", de: 'Die Speichererfassung ist freiwillig und standardmäßig deaktiviert', hi: 'मेमोरी निष्कर्षण वैकल्पिक है और डिफ़ॉल्ट रूप से बंद है', zh: '记忆提取为选择性加入，默认关闭' },
  'auth.why5': { en: 'Refresh-token based silent restoration', ar: 'استعادة صامتة قائمة على رمز التحديث', es: 'Restauración silenciosa basada en token de actualización', fr: 'Restauration silencieuse basée sur un jeton de rafraîchissement', de: 'Stille Wiederherstellung über Refresh-Token', hi: 'रिफ़्रेश-टोकन आधारित मौन पुनर्स्थापना', zh: '基于刷新令牌的静默恢复' },
  'auth.errCreds': { en: 'Enter your email and password to continue.', ar: 'أدخل بريدك الإلكتروني وكلمة المرور للمتابعة.', es: 'Introduce tu correo y contraseña para continuar.', fr: 'Saisissez votre e-mail et votre mot de passe pour continuer.', de: 'Geben Sie zum Fortfahren Ihre E-Mail und Ihr Passwort ein.', hi: 'जारी रखने के लिए अपना ईमेल और पासवर्ड दर्ज करें।', zh: '请输入您的电子邮件和密码以继续。' },
  'auth.errFail': { en: 'We could not sign you in. Please try again.', ar: 'تعذّر تسجيل دخولك. يرجى المحاولة مرة أخرى.', es: 'No pudimos iniciar tu sesión. Inténtalo de nuevo.', fr: 'Nous n\'avons pas pu vous connecter. Veuillez réessayer.', de: 'Anmeldung fehlgeschlagen. Bitte versuchen Sie es erneut.', hi: 'हम आपको साइन इन नहीं कर सके। कृपया पुनः प्रयास करें।', zh: '无法登录。请重试。' },

  // ── Google callback ────────────────────────────────────────
  'gcb.failTitle': { en: 'Sign-in failed', ar: 'فشل تسجيل الدخول', es: 'Error al iniciar sesión', fr: 'Échec de la connexion', de: 'Anmeldung fehlgeschlagen', hi: 'साइन-इन विफल', zh: '登录失败' },
  'gcb.wrong': { en: 'Something went wrong', ar: 'حدث خطأ ما', es: 'Algo salió mal', fr: "Une erreur s'est produite", de: 'Etwas ist schiefgelaufen', hi: 'कुछ गलत हो गया', zh: '出了点问题' },
  'gcb.back': { en: 'Back to sign-in', ar: 'العودة إلى تسجيل الدخول', es: 'Volver al inicio de sesión', fr: 'Retour à la connexion', de: 'Zurück zur Anmeldung', hi: 'साइन-इन पर वापस जाएँ', zh: '返回登录' },
  'gcb.signingin': { en: 'Signing in with Google', ar: 'تسجيل الدخول باستخدام Google', es: 'Iniciando sesión con Google', fr: 'Connexion avec Google', de: 'Anmeldung mit Google', hi: 'Google से साइन इन हो रहा है', zh: '正在使用 Google 登录' },
  'gcb.completing': { en: 'Completing sign-in…', ar: 'جارٍ إكمال تسجيل الدخول…', es: 'Completando el inicio de sesión…', fr: 'Finalisation de la connexion…', de: 'Anmeldung wird abgeschlossen…', hi: 'साइन-इन पूरा हो रहा है…', zh: '正在完成登录…' },
  'gcb.wait': { en: 'Please wait while we verify your Google account.', ar: 'يرجى الانتظار بينما نتحقق من حساب Google الخاص بك.', es: 'Espera mientras verificamos tu cuenta de Google.', fr: 'Veuillez patienter pendant la vérification de votre compte Google.', de: 'Bitte warten Sie, während wir Ihr Google-Konto überprüfen.', hi: 'कृपया प्रतीक्षा करें जबकि हम आपका Google खाता सत्यापित करते हैं।', zh: '请稍候，我们正在验证您的 Google 账户。' },
  'gcb.cancelled': { en: 'Google sign-in was cancelled. You can sign in with email instead.', ar: 'تم إلغاء تسجيل الدخول عبر Google. يمكنك تسجيل الدخول بالبريد الإلكتروني بدلاً من ذلك.', es: 'Se canceló el inicio de sesión con Google. Puedes iniciar sesión con tu correo.', fr: 'La connexion Google a été annulée. Vous pouvez vous connecter par e-mail à la place.', de: 'Die Google-Anmeldung wurde abgebrochen. Sie können sich stattdessen per E-Mail anmelden.', hi: 'Google साइन-इन रद्द कर दिया गया। आप इसके बजाय ईमेल से साइन इन कर सकते हैं।', zh: 'Google 登录已取消。您可以改用电子邮件登录。' },
  'gcb.noCode': { en: 'No authorization code was received from Google. Please try again.', ar: 'لم يُستلم رمز تفويض من Google. يرجى المحاولة مرة أخرى.', es: 'No se recibió ningún código de autorización de Google. Inténtalo de nuevo.', fr: "Aucun code d'autorisation n'a été reçu de Google. Veuillez réessayer.", de: 'Es wurde kein Autorisierungscode von Google empfangen. Bitte versuchen Sie es erneut.', hi: 'Google से कोई प्राधिकरण कोड प्राप्त नहीं हुआ। कृपया पुनः प्रयास करें।', zh: '未从 Google 收到授权码。请重试。' },
  'gcb.noSession': { en: 'Google sign-in succeeded but we could not establish a session. Please try again.', ar: 'نجح تسجيل الدخول عبر Google لكن تعذّر إنشاء جلسة. يرجى المحاولة مرة أخرى.', es: 'El inicio de sesión con Google funcionó, pero no se pudo crear una sesión. Inténtalo de nuevo.', fr: "La connexion Google a réussi mais nous n'avons pas pu établir de session. Veuillez réessayer.", de: 'Die Google-Anmeldung war erfolgreich, aber es konnte keine Sitzung erstellt werden. Bitte versuchen Sie es erneut.', hi: 'Google साइन-इन सफल रहा लेकिन हम सत्र नहीं बना सके। कृपया पुनः प्रयास करें।', zh: 'Google 登录成功，但无法建立会话。请重试。' },
  'gcb.notEnabled': { en: 'Google sign-in is not enabled on this server yet. Please sign in with email.', ar: 'تسجيل الدخول عبر Google غير مُفعّل على هذا الخادم بعد. يرجى تسجيل الدخول بالبريد الإلكتروني.', es: 'El inicio de sesión con Google aún no está habilitado en este servidor. Inicia sesión con tu correo.', fr: "La connexion Google n'est pas encore activée sur ce serveur. Veuillez vous connecter par e-mail.", de: 'Die Google-Anmeldung ist auf diesem Server noch nicht aktiviert. Bitte melden Sie sich per E-Mail an.', hi: 'इस सर्वर पर Google साइन-इन अभी सक्षम नहीं है। कृपया ईमेल से साइन इन करें।', zh: '此服务器尚未启用 Google 登录。请使用电子邮件登录。' },
  'gcb.failGeneric': { en: 'Google sign-in could not be completed. Please try again or use email sign-in.', ar: 'تعذّر إكمال تسجيل الدخول عبر Google. يرجى المحاولة مرة أخرى أو استخدام البريد الإلكتروني.', es: 'No se pudo completar el inicio de sesión con Google. Inténtalo de nuevo o usa el correo.', fr: "La connexion Google n'a pas pu être finalisée. Réessayez ou utilisez la connexion par e-mail.", de: 'Die Google-Anmeldung konnte nicht abgeschlossen werden. Bitte versuchen Sie es erneut oder per E-Mail.', hi: 'Google साइन-इन पूरा नहीं हो सका। कृपया पुनः प्रयास करें या ईमेल साइन-इन का उपयोग करें।', zh: '无法完成 Google 登录。请重试或使用电子邮件登录。' },

  // ── Consent / sign up ──────────────────────────────────────
  'consent.eyebrow': { en: 'Create your account', ar: 'أنشئ حسابك', es: 'Crea tu cuenta', fr: 'Créez votre compte', de: 'Erstellen Sie Ihr Konto', hi: 'अपना खाता बनाएँ', zh: '创建您的账户' },
  'consent.title': { en: 'Terms, privacy, and consent', ar: 'الشروط والخصوصية والموافقة', es: 'Términos, privacidad y consentimiento', fr: 'Conditions, confidentialité et consentement', de: 'Bedingungen, Datenschutz und Einwilligung', hi: 'शर्तें, गोपनीयता और सहमति', zh: '条款、隐私与同意' },
  'consent.lead': {
    en: 'Account creation requires current legal acceptance and explicit controls for processing chat, reminders, attachments, and profile generation.',
    ar: 'يتطلب إنشاء الحساب قبولاً قانونياً حالياً وضوابط صريحة لمعالجة المحادثة والتذكيرات والمرفقات وإنشاء الملف الشخصي.',
    es: 'La creación de la cuenta requiere la aceptación legal vigente y controles explícitos para procesar el chat, los recordatorios, los archivos adjuntos y la generación del perfil.',
    fr: "La création du compte nécessite l'acceptation juridique en vigueur et des contrôles explicites pour le traitement de la discussion, des rappels, des pièces jointes et de la génération du profil.",
    de: 'Die Kontoerstellung erfordert die aktuelle rechtliche Zustimmung und ausdrückliche Kontrollen für die Verarbeitung von Chat, Erinnerungen, Anhängen und Profilerstellung.',
    hi: 'खाता बनाने के लिए वर्तमान कानूनी स्वीकृति और चैट, अनुस्मारक, अनुलग्नक तथा प्रोफ़ाइल निर्माण को संसाधित करने हेतु स्पष्ट नियंत्रण आवश्यक हैं।',
    zh: '创建账户需要当前的法律接受以及对处理对话、提醒、附件和档案生成的明确控制。'
  },
  'consent.terms.title': { en: 'Terms and Conditions', ar: 'الشروط والأحكام', es: 'Términos y condiciones', fr: 'Conditions générales', de: 'Allgemeine Geschäftsbedingungen', hi: 'नियम और शर्तें', zh: '条款与条件' },
  'consent.terms.body': {
    en: 'Version {v}. Users retain control over settings, conversation deletion, and profile extraction.',
    ar: 'الإصدار {v}. يحتفظ المستخدمون بالتحكم في الإعدادات وحذف المحادثات واستخراج الملف الشخصي.',
    es: 'Versión {v}. Los usuarios mantienen el control sobre los ajustes, la eliminación de conversaciones y la extracción del perfil.',
    fr: "Version {v}. Les utilisateurs gardent le contrôle des paramètres, de la suppression des conversations et de l'extraction du profil.",
    de: 'Version {v}. Nutzer behalten die Kontrolle über Einstellungen, das Löschen von Gesprächen und die Profilerfassung.',
    hi: 'संस्करण {v}. उपयोगकर्ता सेटिंग्स, बातचीत हटाने और प्रोफ़ाइल निष्कर्षण पर नियंत्रण बनाए रखते हैं।',
    zh: '版本 {v}。用户保留对设置、对话删除和档案提取的控制权。'
  },
  'consent.terms.point1': { en: 'ORSA is an AI health companion, not a doctor, clinic, emergency service, or medical device.' },
  'consent.terms.point2': { en: 'AI responses may be incomplete, inaccurate, delayed, mistranslated, or unsafe; verify important information with qualified professionals.' },
  'consent.terms.point3': { en: 'For emergencies or rapidly worsening symptoms, contact local emergency services or go to the nearest emergency department.' },
  'consent.terms.point4': { en: 'You are responsible for account security, the content you submit, and making sure you have the right to upload any files.' },
  'consent.terms.point5': { en: 'Uploads, reminders, profile memory, document review, and chat history may be limited, unavailable, or changed as the service evolves.' },
  'consent.terms.point6': { en: 'ORSA may suspend misuse, enforce safety controls, and retain audit records needed for security, compliance, and legal acceptance.' },
  'consent.privacy.title': { en: 'Privacy Policy', ar: 'سياسة الخصوصية', es: 'Política de privacidad', fr: 'Politique de confidentialité', de: 'Datenschutzrichtlinie', hi: 'गोपनीयता नीति', zh: '隐私政策' },
  'consent.privacy.body': {
    en: 'Clinical content, attachments, and profile extraction events are stored with audit trails and service boundaries.',
    ar: 'يُخزّن المحتوى السريري والمرفقات وأحداث استخراج الملف الشخصي مع سجلات تدقيق وحدود للخدمات.',
    es: 'El contenido clínico, los archivos adjuntos y los eventos de extracción del perfil se almacenan con registros de auditoría y límites de servicio.',
    fr: "Le contenu clinique, les pièces jointes et les événements d'extraction de profil sont stockés avec des journaux d'audit et des limites de service.",
    de: 'Klinische Inhalte, Anhänge und Profilerfassungsereignisse werden mit Audit-Protokollen und Service-Grenzen gespeichert.',
    hi: 'क्लिनिकल सामग्री, अनुलग्नक और प्रोफ़ाइल निष्कर्षण घटनाएँ ऑडिट ट्रेल और सेवा सीमाओं के साथ संग्रहीत की जाती हैं।',
    zh: '临床内容、附件和档案提取事件均带有审计记录和服务边界进行存储。'
  },
  'consent.privacy.point1': { en: 'ORSA may process account data, messages, health details, uploads, extracted document text, reminders, settings, consent records, device data, and location data.' },
  'consent.privacy.point2': { en: 'Health-related content may be sent to configured AI/model providers for triage support, document extraction, OCR, vision analysis, and response generation.' },
  'consent.privacy.point3': { en: 'Optional profile memory is controlled by you and should never be used as clinical evidence or to reduce urgency.' },
  'consent.privacy.point4': { en: 'Data may be stored in a database with related records needed for account access, security, auditability, support, legal compliance, and service continuity.' },
  'consent.privacy.point5': { en: 'ORSA does not sell personal health information and should not use it for targeted advertising without a separate legally compliant consent framework.' },
  'consent.privacy.point6': { en: 'You may request access, correction, deletion, export, and withdrawal of optional consent, subject to legal, security, and audit requirements.' },
  'consent.create': { en: 'Create account', ar: 'إنشاء حساب', es: 'Crear cuenta', fr: 'Créer un compte', de: 'Konto erstellen', hi: 'खाता बनाएँ', zh: '创建账户' },
  'consent.acceptTerms': { en: 'I accept the current legal documents.', ar: 'أوافق على المستندات القانونية الحالية.', es: 'Acepto los documentos legales vigentes.', fr: 'J\'accepte les documents juridiques en vigueur.', de: 'Ich akzeptiere die aktuellen rechtlichen Dokumente.', hi: 'मैं वर्तमान कानूनी दस्तावेज़ स्वीकार करता/करती हूँ।', zh: '我接受当前的法律文件。' },
  'consent.memory': {
    en: 'Let ORSA remember helpful details from our conversations to personalize your care. This is optional, and you can turn it off anytime in settings.',
    ar: 'دع ORSA يتذكّر التفاصيل المفيدة من محادثاتنا لتخصيص رعايتك. هذا اختياري، ويمكنك إيقافه في أي وقت من الإعدادات.',
    es: 'Deja que ORSA recuerde detalles útiles de nuestras conversaciones para personalizar tu atención. Es opcional y puedes desactivarlo cuando quieras en los ajustes.',
    fr: "Laissez ORSA mémoriser les détails utiles de nos conversations pour personnaliser vos soins. C'est facultatif et vous pouvez le désactiver à tout moment dans les paramètres.",
    de: 'Lassen Sie ORSA hilfreiche Details aus unseren Gesprächen speichern, um Ihre Betreuung zu personalisieren. Das ist freiwillig und kann jederzeit in den Einstellungen deaktiviert werden.',
    hi: 'ORSA को हमारी बातचीत के उपयोगी विवरण याद रखने दें ताकि आपकी देखभाल को व्यक्तिगत बनाया जा सके। यह वैकल्पिक है, और आप इसे सेटिंग्स में कभी भी बंद कर सकते हैं।',
    zh: '让 ORSA 记住我们对话中的有用细节，以个性化您的护理。这是可选的，您可以随时在设置中关闭。'
  },
  'consent.hint': { en: 'Accept the legal documents above to continue.', ar: 'وافق على المستندات القانونية أعلاه للمتابعة.', es: 'Acepta los documentos legales anteriores para continuar.', fr: 'Acceptez les documents juridiques ci-dessus pour continuer.', de: 'Akzeptieren Sie die obigen rechtlichen Dokumente, um fortzufahren.', hi: 'जारी रखने के लिए ऊपर दिए कानूनी दस्तावेज़ स्वीकार करें।', zh: '请接受上方的法律文件以继续。' },
  'consent.googleSignup': { en: 'Sign up with Google', ar: 'التسجيل باستخدام Google', es: 'Regístrate con Google', fr: "S'inscrire avec Google", de: 'Mit Google registrieren', hi: 'Google के साथ साइन अप करें', zh: '使用 Google 注册' },
  'consent.orEmail': { en: 'or sign up with email', ar: 'أو التسجيل بالبريد الإلكتروني', es: 'o regístrate con tu correo', fr: "ou inscrivez-vous par e-mail", de: 'oder per E-Mail registrieren', hi: 'या ईमेल से साइन अप करें', zh: '或使用电子邮件注册' },
  'consent.creating': { en: 'Creating…', ar: 'جارٍ الإنشاء…', es: 'Creando…', fr: 'Création…', de: 'Wird erstellt…', hi: 'बनाया जा रहा है…', zh: '正在创建…' },
  'consent.savedMsg': { en: 'Account created.', ar: 'تم إنشاء الحساب.', es: 'Cuenta creada.', fr: 'Compte créé.', de: 'Konto erstellt.', hi: 'खाता बन गया।', zh: '账户已创建。' },
  'consent.signinLink': { en: 'Sign in to start chatting', ar: 'سجّل الدخول لبدء المحادثة', es: 'Inicia sesión para empezar a chatear', fr: 'Connectez-vous pour commencer à discuter', de: 'Anmelden, um zu chatten', hi: 'चैट शुरू करने के लिए साइन इन करें', zh: '登录以开始对话' },
  'consent.errFields': { en: 'Enter an email and password to create your account.', ar: 'أدخل بريداً إلكترونياً وكلمة مرور لإنشاء حسابك.', es: 'Introduce un correo y una contraseña para crear tu cuenta.', fr: 'Saisissez un e-mail et un mot de passe pour créer votre compte.', de: 'Geben Sie eine E-Mail und ein Passwort ein, um Ihr Konto zu erstellen.', hi: 'अपना खाता बनाने के लिए ईमेल और पासवर्ड दर्ज करें।', zh: '请输入电子邮件和密码以创建账户。' },
  'consent.errFail': { en: 'We could not create your account. Please try again.', ar: 'تعذّر إنشاء حسابك. يرجى المحاولة مرة أخرى.', es: 'No pudimos crear tu cuenta. Inténtalo de nuevo.', fr: "Nous n'avons pas pu créer votre compte. Veuillez réessayer.", de: 'Ihr Konto konnte nicht erstellt werden. Bitte versuchen Sie es erneut.', hi: 'हम आपका खाता नहीं बना सके। कृपया पुनः प्रयास करें।', zh: '无法创建您的账户。请重试。' },

  // ── Chat ───────────────────────────────────────────────────
  'chat.newConversation': { en: '+ New conversation', ar: '+ محادثة جديدة', es: '+ Nueva conversación', fr: '+ Nouvelle conversation', de: '+ Neues Gespräch', hi: '+ नई बातचीत', zh: '+ 新对话' },
  'chat.search': { en: 'Search', ar: 'بحث', es: 'Buscar', fr: 'Rechercher', de: 'Suchen', hi: 'खोजें', zh: '搜索' },
  'chat.searchPlaceholder': { en: 'Search conversations', ar: 'ابحث في المحادثات', es: 'Buscar conversaciones', fr: 'Rechercher des conversations', de: 'Gespräche durchsuchen', hi: 'बातचीत खोजें', zh: '搜索对话' },
  'chat.brandTag': { en: 'Your Health Companion', ar: 'رفيقك الصحي', es: 'Tu compañero de salud', fr: 'Votre compagnon santé', de: 'Ihr Gesundheitsbegleiter', hi: 'आपका स्वास्थ्य साथी', zh: '您的健康助手' },
  'chat.quota': { en: '{used} / {limit} uploads used today', ar: '{used} / {limit} تحميلات مُستخدمة اليوم', es: '{used} / {limit} cargas usadas hoy', fr: "{used} / {limit} téléversements utilisés aujourd'hui", de: '{used} / {limit} Uploads heute genutzt', hi: 'आज {used} / {limit} अपलोड उपयोग किए गए', zh: '今日已用 {used} / {limit} 次上传' },
  'chat.dropzone': { en: 'Drop files or choose from device', ar: 'أسقط الملفات أو اخترها من جهازك', es: 'Suelta archivos o elige desde el dispositivo', fr: "Déposez des fichiers ou choisissez depuis l'appareil", de: 'Dateien ablegen oder vom Gerät auswählen', hi: 'फ़ाइलें छोड़ें या डिवाइस से चुनें', zh: '拖放文件或从设备中选择' },
  'chat.dropzoneSub': { en: 'PDF, image, camera uploads', ar: 'تحميلات PDF وصور وكاميرا', es: 'Cargas de PDF, imagen y cámara', fr: 'Téléversements PDF, image, caméra', de: 'PDF-, Bild-, Kamera-Uploads', hi: 'PDF, छवि, कैमरा अपलोड', zh: 'PDF、图像、相机上传' },
  'chat.composerPlaceholder': { en: 'Message ORSA', ar: 'راسل ORSA', es: 'Escribe a ORSA', fr: 'Message à ORSA', de: 'Nachricht an ORSA', hi: 'ORSA को संदेश भेजें', zh: '给 ORSA 发消息' },
  'chat.attachmentReviewPrompt': { en: 'Please review the attached files and explain the important findings.', ar: 'يرجى مراجعة الملفات المرفقة وشرح النتائج المهمة.', es: 'Revisa los archivos adjuntos y explica los hallazgos importantes.', fr: 'Veuillez examiner les fichiers joints et expliquer les éléments importants.', de: 'Bitte prüfen Sie die angehängten Dateien und erklären Sie die wichtigen Befunde.', hi: 'कृपया संलग्न फ़ाइलों की समीक्षा करें और महत्वपूर्ण निष्कर्ष समझाएँ।', zh: '请查看附件并解释重要发现。' },
  'chat.send': { en: 'Send', ar: 'إرسال', es: 'Enviar', fr: 'Envoyer', de: 'Senden', hi: 'भेजें', zh: '发送' },
  'chat.aiDisclaimer': { en: 'Orsa is AI and can make mistakes', ar: 'ORSA ذكاء اصطناعي وقد يخطئ.', es: 'ORSA es IA y puede cometer errores.', fr: "ORSA est une IA et peut se tromper.", de: 'ORSA ist KI und kann Fehler machen.', hi: 'ORSA AI है और गलतियाँ कर सकता है।', zh: 'ORSA 是 AI，可能会出错。' },
  'chat.loading': { en: 'ORSA is preparing a response', ar: 'ORSA يجهّز الرد', es: 'ORSA está preparando una respuesta', fr: 'ORSA prépare une réponse', de: 'ORSA bereitet eine Antwort vor', hi: 'ORSA उत्तर तैयार कर रहा है', zh: 'ORSA 正在准备回复' },
  'chat.settings': { en: 'Settings', ar: 'الإعدادات', es: 'Ajustes', fr: 'Paramètres', de: 'Einstellungen', hi: 'सेटिंग्स', zh: '设置' },
  'chat.greeting': { en: 'Tell me what is going on.', ar: 'أخبرني بما يحدث.', es: 'Cuéntame qué te ocurre.', fr: 'Dites-moi ce qui se passe.', de: 'Sagen Sie mir, was los ist.', hi: 'मुझे बताएँ कि क्या हो रहा है।', zh: '告诉我发生了什么。' },
  'chat.uploadLimit': { en: '{n} files queued. Daily limit reached.', ar: '{n} ملفات في قائمة الانتظار. تم بلوغ الحد اليومي.', es: '{n} archivos en cola. Límite diario alcanzado.', fr: '{n} fichiers en file. Limite quotidienne atteinte.', de: '{n} Dateien in der Warteschlange. Tageslimit erreicht.', hi: '{n} फ़ाइलें कतार में। दैनिक सीमा पूरी हुई।', zh: '{n} 个文件已排队。已达每日上限。' },
  'chat.uploadQueued': { en: '{n} file(s) queued.', ar: '{n} ملف(ات) في قائمة الانتظار.', es: '{n} archivo(s) en cola.', fr: '{n} fichier(s) en file.', de: '{n} Datei(en) in der Warteschlange.', hi: '{n} फ़ाइल(ें) कतार में।', zh: '{n} 个文件已排队。' },
  'chat.uploading': { en: 'Uploading attachments...', ar: 'جارٍ تحميل المرفقات...', es: 'Subiendo archivos adjuntos...', fr: 'Téléversement des pièces jointes...', de: 'Anhänge werden hochgeladen...', hi: 'अनुलग्नक अपलोड हो रहे हैं...', zh: '正在上传附件...' },
  'chat.uploaded': { en: '{n} attachment(s) uploaded.', ar: 'تم تحميل {n} مرفق(ات).', es: '{n} archivo(s) adjunto(s) subido(s).', fr: '{n} pièce(s) jointe(s) téléversée(s).', de: '{n} Anhang/Anhänge hochgeladen.', hi: '{n} अनुलग्नक अपलोड हुए।', zh: '已上传 {n} 个附件。' },
  'chat.uploadFail': { en: 'The attachment upload did not complete. Please try again or paste the report values.', ar: 'لم يكتمل تحميل المرفق. يرجى المحاولة مرة أخرى أو لصق قيم التقرير.', es: 'La carga del adjunto no se completó. Inténtalo de nuevo o pega los valores del informe.', fr: "Le téléversement de la pièce jointe n'a pas abouti. Réessayez ou collez les valeurs du rapport.", de: 'Der Anhang wurde nicht vollständig hochgeladen. Bitte erneut versuchen oder die Berichtswerte einfügen.', hi: 'अनुलग्नक अपलोड पूरा नहीं हुआ। कृपया पुनः प्रयास करें या रिपोर्ट मान चिपकाएँ।', zh: '附件上传未完成。请重试或粘贴报告数值。' },
  'chat.sendError': { en: 'I could not reach the triage service just now, so your message was not processed. Please check your connection and try again. If this is an emergency, call your local emergency number.', ar: 'لم أتمكن من الوصول إلى خدمة الفرز الآن، لذلك لم تتم معالجة رسالتك. يرجى التحقق من اتصالك والمحاولة مرة أخرى. إذا كانت هذه حالة طارئة، فاتصل برقم الطوارئ المحلي.', es: 'No pude conectar con el servicio de triaje en este momento, por lo que tu mensaje no se procesó. Comprueba tu conexión e inténtalo de nuevo. Si es una emergencia, llama a tu número de emergencias local.', fr: "Je n'ai pas pu joindre le service de triage pour le moment, votre message n'a donc pas été traité. Vérifiez votre connexion et réessayez. En cas d'urgence, appelez votre numéro d'urgence local.", de: 'Ich konnte den Triage-Dienst gerade nicht erreichen, daher wurde Ihre Nachricht nicht verarbeitet. Bitte prüfen Sie Ihre Verbindung und versuchen Sie es erneut. Im Notfall rufen Sie Ihre örtliche Notrufnummer an.', hi: 'मैं अभी ट्राइएज सेवा तक नहीं पहुँच सका, इसलिए आपका संदेश संसाधित नहीं हुआ। कृपया अपना कनेक्शन जाँचें और पुनः प्रयास करें। यदि यह आपातकाल है, तो अपने स्थानीय आपातकालीन नंबर पर कॉल करें।', zh: '我暂时无法连接分诊服务，因此您的消息未被处理。请检查网络连接后重试。如属紧急情况，请拨打当地急救电话。' },
  'chat.you': { en: 'You', ar: 'أنت', es: 'Tú', fr: 'Vous', de: 'Sie', hi: 'आप', zh: '您' },

  // ── Profile ────────────────────────────────────────────────
  'profile.eyebrow': { en: 'Health profile', ar: 'الملف الصحي', es: 'Perfil de salud', fr: 'Profil de santé', de: 'Gesundheitsprofil', hi: 'स्वास्थ्य प्रोफ़ाइल', zh: '健康档案' },
  'profile.personaSummary': { en: 'Persona summary', ar: 'ملخّص السمات', es: 'Resumen del perfil', fr: 'Résumé du profil', de: 'Profilübersicht', hi: 'पर्सोना सारांश', zh: '画像摘要' },
  'profile.consentStatus': { en: 'Consent status', ar: 'حالة الموافقة', es: 'Estado del consentimiento', fr: 'État du consentement', de: 'Einwilligungsstatus', hi: 'सहमति स्थिति', zh: '同意状态' },
  'profile.lastExtraction': { en: 'Last extraction:', ar: 'آخر استخراج:', es: 'Última extracción:', fr: 'Dernière extraction :', de: 'Letzte Erfassung:', hi: 'अंतिम निष्कर्षण:', zh: '上次提取：' },
  'profile.notRun': { en: 'Not run', ar: 'لم يُنفّذ', es: 'No ejecutado', fr: 'Non exécuté', de: 'Nicht ausgeführt', hi: 'नहीं चला', zh: '未运行' },
  'profile.workflowBoundary': { en: 'Workflow boundary', ar: 'حدود سير العمل', es: 'Límite del flujo de trabajo', fr: 'Limite du flux de travail', de: 'Workflow-Grenze', hi: 'वर्कफ़्लो सीमा', zh: '工作流边界' },
  'profile.boundaryText': { en: 'Stored profile data is not passed into the triage model pipeline.', ar: 'لا تُمرَّر بيانات الملف الشخصي المخزّنة إلى مسار نموذج الفرز.', es: 'Los datos del perfil almacenados no se pasan al flujo del modelo de triaje.', fr: "Les données de profil stockées ne sont pas transmises au pipeline du modèle de triage.", de: 'Gespeicherte Profildaten werden nicht in die Triage-Modell-Pipeline übergeben.', hi: 'संग्रहीत प्रोफ़ाइल डेटा ट्राइएज मॉडल पाइपलाइन में नहीं भेजा जाता।', zh: '存储的档案数据不会传入分诊模型流程。' },
  'profile.status.enabled': { en: 'enabled', ar: 'مُفعّل', es: 'activado', fr: 'activé', de: 'aktiviert', hi: 'सक्षम', zh: '已启用' },
  'profile.status.disabled': { en: 'disabled', ar: 'مُعطّل', es: 'desactivado', fr: 'désactivé', de: 'deaktiviert', hi: 'अक्षम', zh: '已禁用' },

  // ── Settings ───────────────────────────────────────────────
  'profile.consentDesc': { en: 'Allow ORSA to use profile context in each thread.' },
  'profile.summaryPlaceholder': { en: 'Summarize the patient preferences ORSA should remember.' },
  'profile.boundaryPlaceholder': { en: 'Define how profile context may and may not affect the workflow.' },
  'profile.boundaryPrompt': { en: 'AI boundary prompt' },
  'profile.save': { en: 'Save profile' },
  'profile.saved': { en: 'Saved' },

  'settings.backToChat': { en: 'Back to chat', ar: 'العودة إلى المحادثة', es: 'Volver al chat', fr: 'Retour à la discussion', de: 'Zurück zum Chat', hi: 'चैट पर वापस जाएं', zh: '返回对话' },
  'settings.eyebrow': { en: 'Account controls', ar: 'ضوابط الحساب', es: 'Controles de la cuenta', fr: 'Contrôles du compte', de: 'Kontoeinstellungen', hi: 'खाता नियंत्रण', zh: '账户控制' },
  'settings.title': { en: 'Settings', ar: 'الإعدادات', es: 'Ajustes', fr: 'Paramètres', de: 'Einstellungen', hi: 'सेटिंग्स', zh: '设置' },
  'settings.theme': { en: 'Color theme', ar: 'سمة الألوان', es: 'Tema de color', fr: 'Thème de couleur', de: 'Farbthema', hi: 'रंग थीम', zh: '颜色主题' },
  'settings.themeDesc': { en: 'Choose a light or dark look, or match your device.', ar: 'اختر مظهراً فاتحاً أو داكناً، أو طابق جهازك.', es: 'Elige un aspecto claro u oscuro, o adáptalo a tu dispositivo.', fr: "Choisissez un aspect clair ou sombre, ou suivez votre appareil.", de: 'Wählen Sie ein helles oder dunkles Aussehen oder passen Sie es Ihrem Gerät an.', hi: 'हल्का या गहरा रूप चुनें, या अपने डिवाइस से मिलाएँ।', zh: '选择浅色或深色外观，或与您的设备保持一致。' },
  'settings.language': { en: 'App language', ar: 'لغة التطبيق', es: 'Idioma de la app', fr: "Langue de l'application", de: 'App-Sprache', hi: 'ऐप भाषा', zh: '应用语言' },
  'settings.languageDesc': { en: 'Sets the interface language and text direction (Arabic switches to right-to-left).', ar: 'يحدّد لغة الواجهة واتجاه النص (العربية تتحول إلى اليمين لليسار).', es: 'Define el idioma de la interfaz y la dirección del texto (el árabe cambia a derecha a izquierda).', fr: "Définit la langue de l'interface et le sens du texte (l'arabe passe de droite à gauche).", de: 'Legt die Sprache der Oberfläche und die Textrichtung fest (Arabisch wechselt zu rechts-nach-links).', hi: 'इंटरफ़ेस की भाषा और टेक्स्ट दिशा सेट करता है (अरबी दाएँ-से-बाएँ हो जाती है)।', zh: '设置界面语言和文字方向（阿拉伯语切换为从右到左）。' },
  'settings.memory': { en: 'Personalized memory', ar: 'الذاكرة المخصّصة', es: 'Memoria personalizada', fr: 'Mémoire personnalisée', de: 'Personalisierter Speicher', hi: 'व्यक्तिगत मेमोरी', zh: '个性化记忆' },
  'settings.memoryDesc': {
    en: 'Let ORSA remember helpful details from your conversations to personalize your care. This is optional — you can turn it off anytime.',
    ar: 'دع ORSA يتذكّر التفاصيل المفيدة من محادثاتك لتخصيص رعايتك. هذا اختياري — يمكنك إيقافه في أي وقت.',
    es: 'Deja que ORSA recuerde detalles útiles de tus conversaciones para personalizar tu atención. Es opcional: puedes desactivarlo cuando quieras.',
    fr: "Laissez ORSA mémoriser les détails utiles de vos conversations pour personnaliser vos soins. C'est facultatif — désactivable à tout moment.",
    de: 'Lassen Sie ORSA hilfreiche Details aus Ihren Gesprächen speichern, um Ihre Betreuung zu personalisieren. Das ist freiwillig — jederzeit deaktivierbar.',
    hi: 'ORSA को आपकी बातचीत के उपयोगी विवरण याद रखने दें ताकि आपकी देखभाल को व्यक्तिगत बनाया जा सके। यह वैकल्पिक है — आप इसे कभी भी बंद कर सकते हैं।',
    zh: '让 ORSA 记住您对话中的有用细节，以个性化您的护理。这是可选的——您可以随时关闭。'
  },
  'settings.savedOn': { en: 'Personalized memory is on. You can turn it off anytime.', ar: 'الذاكرة المخصّصة مُفعّلة. يمكنك إيقافها في أي وقت.', es: 'La memoria personalizada está activada. Puedes desactivarla cuando quieras.', fr: 'La mémoire personnalisée est activée. Vous pouvez la désactiver à tout moment.', de: 'Personalisierter Speicher ist aktiviert. Sie können ihn jederzeit deaktivieren.', hi: 'व्यक्तिगत मेमोरी चालू है। आप इसे कभी भी बंद कर सकते हैं।', zh: '个性化记忆已开启。您可以随时关闭。' },
  'settings.savedOff': { en: 'Personalized memory is off.', ar: 'الذاكرة المخصّصة مُعطّلة.', es: 'La memoria personalizada está desactivada.', fr: 'La mémoire personnalisée est désactivée.', de: 'Personalisierter Speicher ist deaktiviert.', hi: 'व्यक्तिगत मेमोरी बंद है।', zh: '个性化记忆已关闭。' }
};

/** Translate a key for the given language, with optional {param} interpolation. */
export function translate(lang: LangCode, key: string, params?: Record<string, string | number>): string {
  const entry = TRANSLATIONS[key];
  let value = entry?.[lang] ?? entry?.['en'] ?? key;
  if (params) {
    for (const [name, val] of Object.entries(params)) {
      value = value.replace(new RegExp(`\\{${name}\\}`, 'g'), String(val));
    }
  }
  return value;
}

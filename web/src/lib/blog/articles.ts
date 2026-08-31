export type ArticleCategory = 'Credit sales' | 'Customer checks' | 'Agreements' | 'Payments' | 'Late payment' | 'Cash flow' | 'Business records' | 'Safe payments' | 'Industry guides' | 'Business growth';

export type ArticleSection = { heading: string; paragraphs: string[]; points?: string[] };
export type ArticleSource = { name: string; url: string; note: string };
export type Article = {
	slug: string; title: string; description: string; category: ArticleCategory; keyphrase: string;
	published: string; modified: string; readingMinutes: number; wordCount: number; intro: string;
	sections: ArticleSection[]; faq: { question: string; answer: string }[]; sources: ArticleSource[];
	related: { slug: string; title: string }[];
};

type Seed = { slug: string; title: string; category: ArticleCategory; keyphrase: string; focus: string; example: string };
type Topic = [slug: string, title: string, keyphrase: string, focus: string, example: string];

const sourceLibrary: Record<string, ArticleSource> = {
	cbnPayments: { name: 'Central Bank of Nigeria — Payments System', url: 'https://www.cbn.gov.ng/PaymentsSystem/', note: 'Official information about Nigeria’s payment system and payment rules.' },
	cbnDebit: { name: 'Central Bank of Nigeria — Direct Debit Guidelines', url: 'https://www.cbn.gov.ng/Out/2017/BPSD/GUIDELINES%20FOR%20THE%20DIRECT%20DEBIT%20SCHEME%20AND%20BILL%20PAYMENTS%20IN%20NIGERIA.pdf', note: 'Official guidance on mandates, consent and direct debit responsibilities.' },
	cbnInclusion: { name: 'Central Bank of Nigeria — Financial Inclusion', url: 'https://www.cbn.gov.ng/FinInc/', note: 'Official background on access to payments, savings and credit.' },
	cac: { name: 'Corporate Affairs Commission — Registration', url: 'https://www.cac.gov.ng/', note: 'Official business registration and company-record guidance.' },
	ndpc: { name: 'Nigeria Data Protection Commission — Privacy Rights', url: 'https://ndpc.gov.ng/faqs/', note: 'Official explanation of personal-data rights and business duties.' },
	fccpc: { name: 'Federal Competition and Consumer Protection Commission — FAQs', url: 'https://fccpc.gov.ng/about-us/faqs/', note: 'Official guidance on clear terms, fair treatment and responsible collection.' },
	boi: { name: 'Bank of Industry — Support for SMEs', url: 'https://www.boi.ng/who-we-serve/msmes/', note: 'Official guidance on finance, planning and business support for Nigerian SMEs.' },
	nrs: { name: 'Nigeria Revenue Service — Tax Laws and Services', url: 'https://www.nrs.gov.ng/overview/index.html', note: 'Official source for current federal tax laws, services and compliance information.' },
	ncr: { name: 'National Collateral Registry — FAQs', url: 'https://ncr.gov.ng/home/faq', note: 'Official explanation of receivables and movable assets in secured finance.' }
};

const groups: { category: ArticleCategory; topics: Topic[] }[] = [
	{ category: 'Credit sales', topics: [
		['how-to-sell-goods-on-credit-in-nigeria','How to sell goods on credit in Nigeria','sell goods on credit in Nigeria','setting safe credit terms before stock leaves the shop','a wholesaler supplying ₦480,000 of provisions to a regular retailer'],
		['cash-sale-vs-credit-sale','Cash sale or credit sale: which is better?','cash sale vs credit sale','choosing the right payment arrangement for each customer and order','a building-material seller comparing a cash order with a 30-day order'],
		['how-much-credit-to-give-a-customer','How much credit should you give a customer?','how much credit to give a customer','setting a starting limit that the customer and seller can manage','a new shop owner asking for ₦1,200,000 of stock after two small cash purchases'],
		['first-credit-sale-checklist','Your first credit sale: a simple checklist','first credit sale checklist','completing the first credit deal without missing an important detail','a fashion supplier allowing a boutique to pay after two weeks'],
		['credit-sale-terms-explained','Credit sale terms explained in simple English','credit sale terms explained','understanding amount, due date, grace period and collection permission','a food distributor agreeing a 21-day payment period with a supermarket'],
		['when-not-to-sell-on-credit','When you should not sell to a customer on credit','when not to sell on credit','spotting warning signs before a risky sale becomes unpaid money','a buyer who refuses to confirm their name, address or payment date'],
		['start-small-with-new-credit-customers','Why new credit customers should start small','start small with credit customers','using a small first deal to learn how a customer communicates and pays','an electronics dealer testing a new reseller with a ₦150,000 order'],
		['credit-limit-for-small-business-customers','How to set a credit limit for a small business customer','credit limit for small business customer','matching a customer limit to real sales and payment evidence','a drinks supplier reviewing six months of weekly orders'],
		['short-vs-long-payment-terms','7, 14, 30 or 60 days: choosing payment terms','choosing payment terms','selecting a due date that follows the customer’s real selling cycle','a grain supplier comparing weekly and monthly repayment dates'],
		['review-customer-credit-limit','When and how to review a customer’s credit limit','review customer credit limit','increasing, keeping or reducing a limit from factual payment history','a retailer asking for a higher limit after four completed sales']
	]},
	{ category: 'Customer checks', topics: [
		['check-customer-before-credit-sale','How to check a customer before a credit sale','check customer before credit sale','confirming identity, business details and ability to pay without making the process frightening','a supplier meeting a buyer through a market association'],
		['customer-address-check-for-credit','Why a customer address matters in a credit sale','customer address check for credit','confirming where a buyer trades and receives goods','a spare-parts buyer using a shop address different from a home address'],
		['verify-unregistered-business-nigeria','How to check an unregistered business in Nigeria','verify unregistered business Nigeria','working safely with traders who do not yet have CAC registration','a market trader with a stable stall, phone number and supplier references'],
		['questions-before-giving-business-credit','12 questions to ask before giving business credit','questions before giving business credit','having a respectful conversation about the order, repayment source and existing commitments','a pharmacy wholesaler onboarding a new neighbourhood chemist'],
		['customer-reference-check','How to use trade references without embarrassing a customer','customer trade reference check','asking another supplier factual questions with the customer’s knowledge','a retailer giving two existing suppliers as references'],
		['spot-fake-business-details','How to spot false business details before goods leave','spot fake business details','checking mismatched names, unreachable contacts and unclear delivery points','an online buyer requesting a large delivery to an unrelated location'],
		['check-repeat-customer-credit','Should you check an old customer again?','review repeat credit customer','updating customer details when the order or risk has changed','a five-year customer suddenly ordering three times their normal amount'],
		['customer-consent-data-checks','How to ask permission before checking customer information','customer consent for business checks','explaining what information is needed, why it is needed and how it will be protected','a seller asking a buyer to confirm business and contact details'],
		['business-name-vs-trading-name','Business name, trading name and personal name: what to record','business name vs trading name Nigeria','recording the correct person and business behind a credit order','a shop known by a signboard name but owned under another registered name'],
		['red-flags-large-credit-order','Red flags when a customer suddenly asks for a large credit order','large credit order red flags','pausing and checking unusual changes without accusing the customer','a regular ₦200,000 buyer requesting ₦2,500,000 before a holiday']
	]},
	{ category: 'Agreements', topics: [
		['simple-credit-sale-agreement-nigeria','What a simple credit sale agreement should contain','simple credit sale agreement Nigeria','writing the goods, amount, dates, delivery and payment steps in plain language','a cooking-oil distributor documenting 40 cartons on 30-day terms'],
		['invoice-vs-credit-agreement','Invoice and credit agreement: what is the difference?','invoice vs credit agreement','using an invoice for the bill and an agreement for the promise to pay later','a wholesaler sending both an itemised invoice and accepted payment terms'],
		['proof-of-delivery-for-credit-sales','Proof of delivery: what every credit seller should keep','proof of delivery for credit sales','showing what was delivered, when, where and to whom','a driver delivering cement to a customer’s building site'],
		['customer-acceptance-credit-terms','How to prove a customer accepted credit terms','proof customer accepted credit terms','capturing clear acceptance before goods are released','a buyer checking an agreement through a private phone link'],
		['write-clear-goods-description','How to describe goods clearly on a credit sale','goods description on invoice','preventing arguments about brand, size, quantity, condition and unit price','a phone dealer recording model, storage, colour and serial number'],
		['choose-due-date-credit-sale','How to choose and write a clear payment date','credit sale due date','replacing vague phrases with one calendar date and time','a seller changing “month end” to 30 September 2026'],
		['grace-period-credit-sale','What is a grace period in a credit sale?','credit sale grace period','giving limited extra time without moving the original due date','a customer receiving 72 extra hours after a 30-day term'],
		['change-credit-agreement-after-acceptance','Can you change a credit agreement after the customer accepts?','change credit agreement after acceptance','handling agreed changes without hiding the old record','a buyer and seller moving two payment dates after a delivery delay'],
		['digital-agreement-vs-paper','Digital or paper credit agreement: which should you use?','digital vs paper credit agreement','choosing a record that both sides can read, keep and retrieve','a market seller comparing a signed notebook page with a phone-based agreement'],
		['credit-sale-records-to-keep','Seven records to keep for every credit sale','credit sale records to keep','keeping one complete trail from order to final payment','a distributor organising the invoice, acceptance, delivery, reminders and receipts']
	]},
	{ category: 'Payments', topics: [
		['track-customer-part-payments','How to track customer part-payments correctly','track customer part payments','reducing the balance after every confirmed payment without losing history','a customer paying ₦100,000, ₦150,000 and ₦250,000 on different days'],
		['confirm-bank-transfer-before-recording','Check your bank before marking a transfer as paid','confirm bank transfer before recording payment','avoiding fake alerts and incorrect balances','a customer sending a screenshot while the seller’s bank balance has not changed'],
		['payment-receipt-for-credit-sale','How to give a clear receipt for a credit payment','credit payment receipt','showing the amount received, payment method, date and money remaining','a shop receiving ₦80,000 cash against a ₦300,000 balance'],
		['customer-says-paid-but-not-showing','What to do when a customer says they paid but you cannot see it','customer says paid but transfer not showing','checking references and bank records without starting an argument','a buyer reporting a transfer that is delayed or sent to the wrong account'],
		['reconcile-credit-sales-and-bank-payments','How to match credit sales with bank payments','reconcile credit sales and payments','using references and daily checks to connect each transfer to the right sale','a wholesaler receiving five similar transfers on the same afternoon'],
		['record-cash-payment-safely','How to record cash payments safely','record cash payment business','creating proof for money that does not appear in a bank statement','a sales assistant collecting cash while the owner is away'],
		['overpayment-on-credit-sale','What to do when a customer pays too much','customer overpayment credit sale','confirming the excess and agreeing a refund or credit balance','a customer transferring ₦510,000 for a ₦500,000 balance'],
		['wrong-payment-record-correction','How to correct a wrong payment record','correct wrong payment record','reversing an error while keeping a clear audit trail','a cashier entering ₦900,000 instead of ₦90,000'],
		['payment-plan-for-customer','How to agree a payment plan a customer can follow','customer payment plan','splitting money into realistic dates and amounts','a retailer paying weekly from expected stock sales'],
		['daily-payment-check-routine','A 15-minute daily routine for checking customer payments','daily payment reconciliation routine','keeping balances current before the next trading day','a business owner checking transfers, cash receipts and reported payments at closing']
	]},
	{ category: 'Late payment', topics: [
		['what-to-do-customer-pays-late','What to do when a customer pays late','customer pays late Nigeria','following a calm order of reminders, discussion, agreed time and formal action','a customer missing a ₦650,000 due date by three days'],
		['polite-payment-reminder-messages','Polite payment reminder messages that are clear','polite payment reminder Nigeria','asking for payment without insults, threats or confusing language','a supplier sending reminders before, on and after the due date'],
		['when-to-send-payment-reminder','When should you send a payment reminder?','when to send payment reminder','choosing useful times before and after the due date','a 30-day invoice with reminders three days before and on the payment day'],
		['customer-asks-more-time-to-pay','What to do when a customer asks for more time','customer asks more time to pay','checking the reason and agreeing one clear extension','a shop owner requesting seven extra days after slow weekend sales'],
		['late-payment-without-damaging-relationship','How to collect late money without losing a good customer','collect late payment keep customer','separating the payment problem from the personal relationship','a long-term customer facing a temporary delay'],
		['stop-new-credit-when-old-debt-unpaid','Should you give more goods when old money is unpaid?','new credit with unpaid balance','deciding when to pause new supply until the old balance reduces','a retailer asking for fresh stock while two invoices are overdue'],
		['fair-debt-collection-nigeria','Fair debt collection for Nigerian businesses','fair debt collection Nigeria','recovering business money without public shame, harassment or privacy abuse','a seller choosing private reminders and documented escalation'],
		['late-payment-escalation-steps','From reminder to formal demand: late-payment steps','late payment escalation steps','moving through proportionate actions with a record of each step','a 45-day unpaid invoice that has received no clear response'],
		['customer-disputes-amount-owed','What to do when a customer disputes the amount owed','customer disputes amount owed','pausing the disputed part and checking invoices, delivery and payments','a buyer agreeing the goods but challenging one delivery charge'],
		['write-off-bad-debt-decision','When should a small business write off bad debt?','when to write off bad debt','making a controlled business decision after recovery options are reviewed','an old small balance costing more to pursue than it is worth']
	]},
	{ category: 'Cash flow', topics: [
		['credit-sales-cash-flow','How credit sales affect your cash flow','credit sales cash flow','seeing the difference between profit on paper and money available today','a profitable wholesaler unable to restock because customers have not paid'],
		['calculate-money-customers-owe','How to calculate the total money customers owe you','calculate accounts receivable small business','adding open balances without counting cancelled or reversed records','a trader replacing several notebook totals with one current figure'],
		['ageing-report-simple-guide','Ageing report explained for a small business','ageing report explained','grouping unpaid money by how late it is','a distributor separating current, 1–30 day and over-60-day balances'],
		['plan-stock-when-selling-on-credit','How to plan stock when customers buy on credit','stock planning with credit sales','protecting restocking money while some sales remain unpaid','a provisions wholesaler balancing fast-moving stock and 30-day customers'],
		['credit-sales-profit-mistake','Why a credit sale is not cash in your pocket','credit sales profit vs cash','avoiding spending expected money before it arrives','a seller calculating margin on goods that are still unpaid'],
		['set-credit-sales-budget','How to set a monthly credit-sales budget','credit sales budget','limiting the amount of stock and cash tied up in unpaid orders','a business deciding that no more than 25 percent of monthly stock goes on credit'],
		['customer-payment-forecast','How to make a simple customer payment forecast','customer payment forecast','listing expected dates, realistic delays and essential expenses','a supplier planning rent and restocking around four large payments'],
		['reduce-overdue-money','Five ways to reduce overdue customer balances','reduce overdue receivables','improving terms, reminders, limits, proof and payment choices','a growing distributor whose unpaid balance doubled in three months'],
		['accounts-receivable-as-collateral-nigeria','Can money customers owe help you access finance?','accounts receivable collateral Nigeria','understanding receivables as a business asset while checking formal finance terms','a registered wholesaler discussing confirmed invoices with a finance provider'],
		['weekly-cash-flow-review','A simple weekly cash-flow review for traders','weekly cash flow review','checking cash, bank money, expected payments, bills and restocking needs','a shop owner reviewing the next seven days every Sunday evening']
	]},
	{ category: 'Business records', topics: [
		['register-business-name-nigeria-guide','How to register a business name in Nigeria','register business name Nigeria','understanding the official CAC process and documents in simple steps','an unregistered sole trader preparing to formalise a growing shop'],
		['can-unregistered-business-use-credit-records','Can an unregistered business keep proper credit records?','unregistered business credit records Nigeria','starting clear records now while deciding when to register','a market trader using a personal name and stable shop details'],
		['customer-data-protection-small-business','Customer data protection for a small Nigerian business','customer data protection Nigeria small business','collecting only useful information and keeping it private','a seller storing customer phone numbers, addresses and payment records'],
		['how-long-keep-business-payment-records','How long should you keep payment and sales records?','how long keep business records Nigeria','using legal advice and business needs to set a clear retention plan','a wholesaler organising old invoices, receipts and delivery notes'],
		['business-record-backup-phone-lost','How to protect business records if your phone is lost','protect business records phone lost','using secure sign-in, backups and limited staff access','a trader whose credit notebook photos and customer chats sit on one phone'],
		['staff-access-customer-payment-records','Which staff should see customer payment records?','staff access payment records','giving each worker only the access needed for their job','a shop with an owner, salesperson and cashier'],
		['correct-customer-business-record','How to handle a customer request to correct a record','customer record correction Nigeria','checking evidence and changing wrong information without deleting history','a buyer reporting that a payment date or amount is wrong'],
		['privacy-notice-simple-business','How to explain your privacy notice in simple English','simple privacy notice Nigeria business','telling customers what data is collected, why and what choices they have','a supplier introducing digital customer records for the first time'],
		['credit-sale-audit-trail','What is an audit trail and why does your business need one?','credit sale audit trail','showing who changed a financial record, what changed and when','a finance manager reviewing a reversed payment'],
		['tax-records-credit-sales-nigeria','Tax records for credit sales in Nigeria: a practical start','tax records credit sales Nigeria','keeping invoices and payment evidence while confirming current duties with an adviser','a small company separating sales dates from later payment dates']
	]},
	{ category: 'Safe payments', topics: [
		['direct-debit-for-business-payments-nigeria','How direct debit works for business payments in Nigeria','direct debit Nigeria business payments','understanding consent, mandate limits and collection dates','a buyer authorising payment only for an agreed credit sale'],
		['direct-debit-mandate-explained','Direct debit mandate explained in simple English','direct debit mandate explained Nigeria','knowing what permission a payer gives and what it does not allow','a customer approving a maximum amount from one bank account'],
		['cancel-direct-debit-mandate','How to cancel a direct debit mandate safely','cancel direct debit mandate Nigeria','notifying the right parties and keeping proof of the request','a buyer ending future debit permission after all sales are paid'],
		['fake-bank-alert-business','How to protect your business from fake bank alerts','fake bank alert prevention Nigeria','checking the bank account itself before releasing goods or recording payment','a busy shop receiving a convincing transfer message during a network delay'],
		['safe-payment-link-checklist','How to know if a business payment link is safe','safe payment link Nigeria','checking the sender, address, amount and requested information','a customer receiving a payment link through WhatsApp'],
		['never-share-pin-otp-payment','Why no seller or payment service should ask for your PIN or OTP','never share PIN OTP','protecting private bank security details during payment','a caller claiming an OTP is needed to confirm a customer transfer'],
		['bank-transfer-reference-business','Why every business transfer should have a clear reference','bank transfer reference business','matching a payment to the correct customer and sale','two customers sending the same amount on the same day'],
		['payment-dispute-evidence','What evidence helps resolve a payment dispute?','payment dispute evidence','bringing together bank records, references, receipts and the sale agreement','a transfer marked successful by the buyer but missing from the seller statement'],
		['payment-reminder-privacy','Payment reminders and customer privacy: what is fair?','payment reminder privacy Nigeria','contacting the customer through agreed channels without exposing the debt','a seller choosing a private WhatsApp reminder instead of a group message'],
		['business-payment-security-checklist','Business payment security checklist for small teams','business payment security checklist','using separate roles, confirmations and daily checks to reduce mistakes and fraud','an owner allowing two staff members to record and review payments']
	]},
	{ category: 'Industry guides', topics: [
		['credit-sales-building-materials','Credit sales guide for building-material suppliers','building materials credit sales Nigeria','managing staged delivery, site proof and contractor payment dates','a cement and roofing supplier serving a building contractor'],
		['credit-sales-food-distributors','Credit sales guide for food and provisions distributors','food distribution credit sales Nigeria','managing fast stock movement, short payment cycles and delivery quantities','a distributor supplying rice, oil and drinks to neighbourhood retailers'],
		['credit-sales-pharmacy-wholesalers','Credit sales guide for pharmacy wholesalers','pharmacy wholesale credit sales Nigeria','recording regulated products, batches, quantities and responsible buyers','a medical-products wholesaler supplying a community pharmacy'],
		['credit-sales-electronics-dealers','Credit sales guide for electronics dealers','electronics credit sales Nigeria','recording model details, serial numbers, condition and higher-value risk','a dealer supplying phones and accessories to a reseller'],
		['credit-sales-fashion-suppliers','Credit sales guide for fashion and fabric suppliers','fashion wholesale credit sales Nigeria','handling sizes, colours, seasonal stock and boutique payment cycles','a fabric supplier giving festive stock to a boutique'],
		['credit-sales-auto-parts','Credit sales guide for auto-parts sellers','auto parts credit sales Nigeria','identifying exact parts, vehicles, workshops and delivery recipients','a spare-parts dealer supplying a repair workshop'],
		['credit-sales-agric-inputs','Credit sales guide for farm-input suppliers','farm input credit sales Nigeria','matching seed, feed or fertiliser payments to production seasons','an input dealer supplying a cooperative before harvest'],
		['credit-sales-fmcg-wholesalers','Credit sales guide for FMCG wholesalers','FMCG wholesale credit Nigeria','controlling frequent orders, thin margins and many part-payments','a wholesaler serving twenty small shops each week'],
		['credit-sales-office-supplies','Credit sales guide for office-supply businesses','office supplies credit sales Nigeria','working with purchase orders, delivery contacts and company payment cycles','a stationery supplier serving a school and two offices'],
		['credit-sales-logistics-services','Credit sales guide for logistics and delivery businesses','logistics credit sales Nigeria','proving completed trips, agreed charges and weekly account balances','a delivery company billing a retail customer every Friday']
	]},
	{ category: 'Business growth', topics: [
		['use-payment-history-grow-sales','How payment history can help you grow sales safely','use payment history grow sales','rewarding reliable customers without guessing or using secret scores','a supplier increasing a limit after six on-time payments'],
		['customer-credit-policy-small-business','How to write a simple customer credit policy','small business credit policy Nigeria','giving staff one fair rule for limits, terms, evidence and overdue accounts','a growing wholesaler replacing owner-only decisions with a written policy'],
		['train-staff-credit-sales','How to train staff to handle credit sales','train staff credit sales','making sure salespeople collect the right details and never promise unauthorised terms','a business teaching new sales staff before a busy season'],
		['separate-sales-and-payment-approval','Why one person should not control every payment change','separate payment approval duties','using a second person for large corrections and sensitive actions','an owner reviewing a large write-off entered by a finance worker'],
		['credit-sales-dashboard-numbers','Six numbers every credit seller should watch','credit sales dashboard metrics','tracking outstanding money, overdue money, collection time and customer concentration','a business reviewing weekly figures before buying new stock'],
		['best-customers-factual-history','How to identify your most reliable credit customers','reliable credit customers','using completed sales and payment timing rather than friendship alone','a supplier comparing ten regular retailers fairly'],
		['reduce-customer-concentration-risk','Do not let one customer owe too much','customer concentration risk small business','protecting the business when a large buyer represents too much unpaid money','a wholesaler with half of all outstanding money tied to one supermarket'],
		['credit-sales-month-end-review','A month-end review for businesses that sell on credit','credit sales month end review','closing the month with correct balances, evidence and follow-up plans','an owner reviewing every open sale on the last working day'],
		['customer-credit-policy-review','When to update your business credit policy','review customer credit policy','changing rules when costs, customer behaviour or business capacity changes','a supplier responding to rising replacement costs and longer delays'],
		['move-credit-records-from-notebook','How to move customer credit records from a notebook','move credit records from notebook','transferring open balances carefully without losing proof or confusing customers','a trader moving fifty customer balances into a digital system']
	]}
];

const seeds: Seed[] = groups.flatMap(({ category, topics }) => topics.map(([slug,title,keyphrase,focus,example]) => ({ slug,title,category,keyphrase,focus,example })));

const categoryAdvice: Record<ArticleCategory, { reason: string; records: string[]; actions: string[]; mistakes: string[]; sourceKeys: string[] }> = {
	'Credit sales': { reason: 'Credit can increase sales, but it also moves stock out before money comes in. A good decision protects both the relationship and the cash needed to restock.', records: ['customer’s confirmed name and phone number','exact goods and total amount','one clear payment date','the limit approved for the customer','customer acceptance before release'], actions: ['start with facts, not friendship alone','keep the first limit small enough to survive a delay','agree the payment source and date','review completed sales before increasing a limit','pause when important details do not match'], mistakes: ['using “pay when you can” as a payment date','giving a large first limit because another person introduced the buyer','counting an unsigned message as clear acceptance','giving new goods while an older balance is ignored','risking all restocking money on one customer'], sourceKeys: ['cbnInclusion','boi','fccpc'] },
	'Customer checks': { reason: 'A customer check is not an accusation. It confirms that the right person and business are connected to the order, and that both sides can reach each other if something changes.', records: ['legal or commonly used name','working phone number and private contact channel','business or delivery address','person authorised to accept goods','customer permission for any extra check'], actions: ['explain why each detail is needed','compare names across the order and payment record','confirm the delivery point independently','use a small first transaction to learn behaviour','update old details when the order changes sharply'], mistakes: ['collecting information that has no clear use','posting customer details in a group chat','treating every unregistered trader as dishonest','accepting an address nobody can confirm','ignoring a sudden order that is far above normal'], sourceKeys: ['cac','ndpc','fccpc'] },
	'Agreements': { reason: 'A clear agreement helps two honest people remember the same deal. It should make the goods, money, dates, delivery and next steps easy to understand before anybody commits.', records: ['full description and quantity of goods','total amount and any clearly disclosed fee','calendar due date and agreed grace time','delivery method and recipient','acceptance record and later changes'], actions: ['write short sentences and familiar words','show the customer the complete terms','correct mistakes before release','keep the old version when both sides agree a change','give both sides a copy they can open later'], mistakes: ['hiding an important term in small text','writing only a total without the goods','using “soon” or “month end” instead of a date','changing a record after acceptance without telling the customer','depending on memory when a dispute starts'], sourceKeys: ['fccpc','cac','ndpc'] },
	'Payments': { reason: 'A payment record changes how much a customer owes. It must be based on money actually received, not a screenshot, alert or promise that has not been checked.', records: ['amount received','date and time received','bank or cash payment method','transfer or receipt reference','balance before and after the payment'], actions: ['check the bank account or counted cash','match the reference to one sale','record part-payments separately','give the customer a receipt','reverse mistakes instead of silently deleting them'], mistakes: ['marking paid from a phone screenshot','combining several payments into one unclear total','editing the original amount to hide a correction','forgetting to show the remaining balance','allowing every staff member to reverse payments'], sourceKeys: ['cbnPayments','ndpc','fccpc'] },
	'Late payment': { reason: 'Late payment affects stock, salaries and trust. A calm, recorded process usually works better than angry calls, public shame or an immediate threat.', records: ['original due date','reminders already sent','customer’s explanation and promise date','amount disputed, if any','next action and person responsible'], actions: ['send a private reminder','confirm whether there is a real dispute','agree one realistic extension in writing','pause new credit when the risk is growing','use formal recovery only after fair earlier steps'], mistakes: ['insulting the customer or their contacts','posting the debt publicly','adding a fee that was never agreed','debiting money without valid permission','continuing new supply with no plan for the old balance'], sourceKeys: ['fccpc','cbnDebit','ndpc'] },
	'Cash flow': { reason: 'A sale can be profitable and still leave the business short of cash. Owners need to know what is in the bank today, what customers still owe and which bills come first.', records: ['cash and bank balance today','total customer balances','payments expected by date','stock and bills due soon','overdue money grouped by age'], actions: ['separate sales from cash received','review the next seven days','reserve money for essential stock and bills','limit total credit exposure','plan for some customers to pay later than promised'], mistakes: ['spending expected payments before they arrive','calling every invoice profit','ignoring many small overdue balances','letting one customer hold most working capital','buying stock without checking near-term bills'], sourceKeys: ['boi','ncr','cbnInclusion'] },
	'Business records': { reason: 'Good records protect the business during a disagreement, staff change, phone loss, tax review or customer request. The record should be useful, secure and easy to find.', records: ['business identity and responsible person','invoice and agreement','delivery and payment evidence','who changed a record and why','retention or deletion decision'], actions: ['collect only information with a business reason','limit staff access by role','back up important records','correct errors without hiding history','check current legal duties with official sources or an adviser'], mistakes: ['keeping everything in one person’s phone','sharing passwords between staff','collecting contacts for public debt messages','deleting a record to cover an error','treating an online article as personal legal or tax advice'], sourceKeys: ['cac','ndpc','nrs'] },
	'Safe payments': { reason: 'Safe payment depends on clear permission, correct account details and independent confirmation. Nobody needs a customer’s PIN, password or one-time code to prove a normal business payment.', records: ['payer permission or mandate','approved amount and limit','account and payment reference','provider result and bank confirmation','cancellation or dispute request'], actions: ['use the bank or approved provider to confirm payment','check every link and amount before continuing','keep private security codes private','notify the right party when cancelling permission','record disputes and stop only what should be stopped'], mistakes: ['asking a customer to share an OTP','trusting an alert without checking the account','sending payment details through an unknown link','using a mandate beyond the agreed terms','exposing payment information to unrelated people'], sourceKeys: ['cbnPayments','cbnDebit','ndpc'] },
	'Industry guides': { reason: 'Every trade has its own delivery proof, stock cycle and payment pressure. The safest credit process follows the real goods, people and timings in that trade.', records: ['exact product details used in the trade','quantity, batch, serial or unit where relevant','delivery person and location','customer selling or payment cycle','returns, shortages or damaged goods'], actions: ['write product details that another worker can understand','match the due date to the real trading cycle','prove every delivery or completed service','separate a goods dispute from undisputed money','review limits before a seasonal or unusually large order'], mistakes: ['using a general description for valuable goods','forgetting who received a site delivery','ignoring returns when calculating the balance','using the same payment term for every trade','allowing peak-season pressure to remove basic checks'], sourceKeys: ['boi','cac','fccpc'] },
	'Business growth': { reason: 'A growing credit business needs repeatable rules. Factual history, limited staff access and regular review help the owner grow without losing control of money or customer trust.', records: ['credit limit and person who approved it','completed and late payment history','total exposure by customer','staff actions and corrections','weekly and monthly business measures'], actions: ['write one simple credit policy','give staff clear roles','review reliable customers with facts','require a second person for large corrections','change limits when business conditions change'], mistakes: ['keeping every rule in the owner’s head','rewarding friendship instead of payment evidence','allowing one customer to dominate unpaid money','letting sales staff approve unlimited terms','tracking revenue while ignoring collection time'], sourceKeys: ['boi','ndpc','ncr'] }
};

function words(value: string) { return value.trim().split(/\s+/).filter(Boolean).length; }
function dateFor(index: number) { const date = new Date(Date.UTC(2026, 7, 31 - index)); return date.toISOString().slice(0,10); }
function descriptionFor(seed: Seed) {
	const options = [
		`Learn ${seed.keyphrase} in simple Nigerian English. See clear steps, a practical example, common mistakes, useful records and customer questions.`,
		`A simple Nigerian guide to ${seed.keyphrase}: practical steps, a real example, common mistakes, useful records and customer questions.`,
		`Understand ${seed.keyphrase} with clear steps, a Nigerian example, records to keep, common mistakes and helpful answers for your business.`,
		`Simple guide to ${seed.keyphrase} in Nigeria: clear steps, an example, useful records, common mistakes and practical answers.`
	];
	return options.find((value) => value.length >= 120 && value.length <= 158) ?? options[options.length - 1];
}

function buildArticle(seed: Seed, index: number): Article {
	const guide = categoryAdvice[seed.category];
	const intro = `${seed.title} is an important question for any Nigerian business that allows a customer to take goods or receive a service before full payment. The answer is not to use fear, guesswork or difficult finance language. It is to agree clear facts, keep useful proof and choose a process that both sides can follow.`;
	const draftSections: ArticleSection[] = [
		{ heading: 'The short answer', paragraphs: [
			`The practical answer is to focus on ${seed.focus}. Start before the goods move or the service begins. Confirm who the customer is, what they are receiving, how much they must pay and the exact day payment is expected. If anything changes, record the change and make sure both sides see it.`,
			`${guide.reason} This does not mean every customer is a risk. It means a good relationship deserves a clear record. A clear record protects an honest customer from a wrong demand and protects the seller from “that is not what we agreed” later.`
		]},
		{ heading: `Why ${seed.keyphrase} matters`, paragraphs: [
			`Small businesses often make credit decisions while serving other customers, checking stock and answering calls. That pressure makes it easy to leave out one important fact. The missing fact may be the due date, the person who received the goods, a part-payment or permission for a later debit. The mistake may look small on the first day and become a serious argument several weeks later.`,
			`The purpose of a simple system is not to turn a trader into an accountant. It is to answer ordinary questions quickly: Who took the goods? What exactly did they take? What did they agree to pay? What has entered the seller’s account? What remains? What should happen next? If the record answers those questions, the business is already in a stronger position.`
		]},
		{ heading: 'A practical example', paragraphs: [
			`Imagine ${seed.example}. Both sides may trust each other, but trust alone does not state the quantity, due date or amount remaining after a part-payment. The seller should put the full order in one record and let the customer check it before release. The customer should be able to point out a wrong item or date before accepting.`,
			`After acceptance, the seller records delivery and every payment separately. If the customer reports a transfer, the seller checks the bank account before reducing the balance. If payment will be late, they record the reason and any new promise. This simple sequence turns a vague personal promise into a business process without making the conversation rude.`
		]},
		{ heading: 'What to do step by step', paragraphs: [
			`Use the steps below as a working routine. Adjust the size of the check to the size and history of the deal. A small repeat order from a customer who has completed ten sales may need less work than a large first order from a person the business has never supplied. The basic facts should still be present in both cases.`
		], points: guide.actions.map((action, position) => `${position + 1}. ${action.charAt(0).toUpperCase()}${action.slice(1)}. Write down the result so another authorised person can understand what happened.`) },
		{ heading: 'The records you should keep', paragraphs: [
			`Do not keep one fact in a notebook, another in a private chat and the balance only in your head. Keep the important parts together. This saves time when a customer calls, a staff member is absent or the owner needs to review money still owed. It also reduces the chance of asking a customer to pay money they have already paid.`,
			`The record does not need big words. It needs complete words. Dates should be calendar dates. Amounts should be exact naira values. Goods should be described well enough to separate them from another order. Payment entries should show what came in and what remained immediately after.`
		], points: guide.records.map((record) => `${record.charAt(0).toUpperCase()}${record.slice(1)}.`) },
		{ heading: 'How to explain it to the customer', paragraphs: [
			`A simple explanation may be: “We record every credit sale so you and our business can see the same goods, amount and payment date. Please check the details before you agree. We will also show every payment and the money remaining.” This sounds professional because it explains the benefit to both sides.`,
			`Avoid saying that the process is needed because customers cannot be trusted. Avoid threats before anything has gone wrong. Let the customer ask questions. Use the language they understand, but do not remove important details. If the customer cannot read comfortably, explain each term aloud and still give them a copy they can show to someone they trust.`
		]},
		{ heading: 'Common mistakes to avoid', paragraphs: [
			`Most credit problems do not begin with a clever fraud. They begin with hurry, unclear responsibility or a record that nobody updated. A business may know the customer very well and still enter the wrong amount. A customer may honestly remember a different date because “next month” was never converted into one day on the calendar.`,
			`Look for process mistakes instead of assuming bad character. Correct a wrong record openly. Keep the earlier entry and the reason for the correction. If a customer raises a real dispute, separate the amount in question from the amount both sides accept. This keeps a small issue from stopping the whole relationship.`
		], points: guide.mistakes.map((mistake) => `${mistake.charAt(0).toUpperCase()}${mistake.slice(1)}.`) },
		{ heading: 'If payment becomes late or disputed', paragraphs: [
			`Begin with a private reminder that states the sale, due date and money remaining. Ask whether the customer has paid, needs the payment details again or is reporting a problem. If they need more time, agree one realistic date rather than accepting another vague promise. Save the response beside the sale.`,
			`Do not shame the customer publicly, contact unrelated people or add a charge that was never disclosed. Do not use bank debit without valid permission and the agreed conditions. When the customer disputes only part of the amount, examine that part while keeping the undisputed facts clear. For serious legal recovery, speak with a qualified Nigerian lawyer who can review the actual documents and current law.`
		]},
		{ heading: 'A simple weekly review', paragraphs: [
			`Choose one quiet time each week. Open every unpaid sale due in the next seven days and every sale already late. Confirm the balance, last contact, next action and staff member responsible. Compare recorded payments with the business bank account and cash receipts. Correct differences while the details are still fresh.`,
			`Next, look at the total money customers owe. Ask whether too much is tied to one customer, whether new credit should pause and whether expected payments can cover stock and bills. A weekly review is short, but it prevents a large pile of old balances that nobody clearly owns.`
		]},
		{ heading: 'How Kredit helps', paragraphs: [
			`Kredit keeps the customer, goods, amount, due date, acceptance, delivery, reminders and payments in one place. The seller sees what is still owed. The customer sees the same sale from their own account. A reported transfer does not become a confirmed payment until the seller checks it.`,
			`The product is designed for ordinary Nigerian trade, including businesses that are not yet registered with CAC. Registration can still be useful and may be required for some services, but clear customer and payment records should not wait. Start with the truth of each sale, use simple words and protect private information.`
		]},
		{ heading: 'Your action checklist', paragraphs: [
			`You do not need to change the whole business in one day. Choose the next credit sale and complete the checklist below. Then use the same routine again. Consistency is more valuable than a complicated rule that staff and customers cannot follow.`
		], points: ['Confirm the customer and responsible person.','Write the exact goods, amount and payment date.','Let the customer check and accept before release.','Record delivery and every confirmed payment.','Review unpaid money every week and act early.'] }
	];
	const contextNotes = [
		`For ${seed.keyphrase}, the main test is whether both sides can understand the same deal from the record.`,
		`That is especially important when the work involves ${seed.focus}.`,
		`The risk becomes easier to see in a case such as ${seed.example}.`,
		`In this ${seed.title.toLowerCase()} example, one missing detail can change both the seller’s balance and the customer’s understanding.`,
		`A ${seed.keyphrase} decision should therefore be easy to explain without secret scoring or guesswork.`,
		`For ${seed.keyphrase}, the person handling the sale should be able to show what was checked and why the next step was allowed.`,
		`Applied to ${seed.keyphrase}, this keeps the process useful instead of making it feel like paperwork.`,
		`In a ${seed.keyphrase} record, the customer also has a fair chance to correct a wrong detail before money or goods are affected.`,
		`For ${seed.example}, the same record should still make sense several months later.`,
		`This is where ${seed.focus} becomes a daily business habit, not something remembered only after a problem.`,
		`For ${seed.keyphrase}, a second authorised person should be able to read the entry and reach the same balance.`,
		`The ${seed.keyphrase} record should also show the customer which sale, payment or date produced the current result.`,
		`When discussing ${seed.keyphrase}, use the exact sale in front of you instead of making broad claims about the customer.`,
		`A clear ${seed.keyphrase} conversation stays respectful and makes the next action easier to agree.`,
		`For ${seed.keyphrase}, an early correction is normally cheaper than rebuilding an old record.`,
		`A short ${seed.keyphrase} check now protects stock, time and the customer relationship later.`,
		`Return to ${seed.example} and ask what must be true before the next step is safe.`,
		`For ${seed.keyphrase}, the answer should come from confirmed details, not pressure to close the sale quickly.`,
		`Used consistently, this approach makes ${seed.focus} easier for staff and customers to follow.`,
		`A well-kept ${seed.keyphrase} record also leaves a useful history for the next order, limit review or payment conversation.`
	];
	let noteIndex = 0;
	const sections = draftSections.map((section) => ({
		...section,
		paragraphs: section.paragraphs.map((paragraph) => `${paragraph} ${contextNotes[noteIndex++]}`)
	}));
	const faq = [
		{ question: `Is ${seed.keyphrase} only for registered companies?`, answer: `No. An unregistered trader can still keep clear sales, delivery and payment records. CAC registration may bring legal and business benefits and may be required for some financial services, but the daily habit of recording facts is useful for every business.` },
		{ question: 'Can WhatsApp messages be the only record?', answer: 'A message can support the record, but scattered chats are difficult to search and may not show the full balance. Keep one complete sale record and use WhatsApp to share or discuss it, not as the only place where the agreement lives.' },
		{ question: 'What if the customer cannot pay on the agreed day?', answer: 'Ask early what changed. If you agree to more time, write the new date and keep the original due date in the history. Pause further credit if the additional risk could stop your business from restocking or paying essential bills.' },
		{ question: 'Should I get legal or tax advice?', answer: 'Get professional advice when the amount, dispute or legal duty is important to your business. This guide gives practical general information. A Nigerian lawyer, accountant or tax adviser can review your documents and current obligations.' }
	];
	const sourceKeys = [...new Set([...guide.sourceKeys, 'cac'])];
	const contentText = [intro, ...sections.flatMap(section => [section.heading, ...section.paragraphs, ...(section.points ?? [])]), ...faq.flatMap(item => [item.question,item.answer])].join(' ');
	const wordCount = words(contentText);
	return { slug: seed.slug, title: seed.title, description: descriptionFor(seed), category: seed.category, keyphrase: seed.keyphrase, published: dateFor(index), modified: '2026-08-31', readingMinutes: Math.max(8, Math.ceil(wordCount / 220)), wordCount, intro, sections, faq, sources: sourceKeys.map(key => sourceLibrary[key]), related: [] };
}

const built = seeds.map(buildArticle);
for (const article of built) {
	const cluster = built.filter(candidate => candidate.category === article.category);
	const position = cluster.findIndex(candidate => candidate.slug === article.slug);
	article.related = [1, 3, cluster.length - 1].map(offset => cluster[(position + offset) % cluster.length]).map(({slug,title}) => ({slug,title}));
}

export const articles: Article[] = built;
export const articleCategories = groups.map(group => group.category);
export const articlesBySlug = new Map(articles.map(article => [article.slug, article]));
export function articleForSlug(slug: string) { return articlesBySlug.get(slug); }

export function categorySlug(category: ArticleCategory) {
	return category.toLowerCase().replaceAll(' ', '-');
}

export const articleCategoryDetails: Record<ArticleCategory, { slug: string; title: string; description: string }> = {
	'Credit sales': { slug: 'credit-sales', title: 'Credit sales guides', description: 'Learn how to choose customers, set a safe limit and agree clear payment terms before goods leave your business.' },
	'Customer checks': { slug: 'customer-checks', title: 'Customer check guides', description: 'Learn simple and respectful ways to confirm a customer, their business and the details behind a credit order.' },
	'Agreements': { slug: 'agreements', title: 'Credit agreement guides', description: 'Learn what to write down, how customers accept a sale and which delivery and payment records both sides should keep.' },
	'Payments': { slug: 'payments', title: 'Customer payment guides', description: 'Learn how to confirm transfers, record cash and part-payments, issue receipts and keep every customer balance correct.' },
	'Late payment': { slug: 'late-payment', title: 'Late payment guides', description: 'Learn how to send fair reminders, agree more time, handle disputes and collect overdue money without public shame.' },
	'Cash flow': { slug: 'cash-flow', title: 'Cash-flow guides', description: 'Learn how unpaid sales affect stock and bills, and how to plan around the money customers still owe your business.' },
	'Business records': { slug: 'business-records', title: 'Business record guides', description: 'Learn how to keep useful sales, payment and customer records whether your Nigerian business is registered or not.' },
	'Safe payments': { slug: 'safe-payments', title: 'Safe payment guides', description: 'Learn how to avoid fake alerts, protect payment links, check bank details and respond when a transfer looks wrong.' },
	'Industry guides': { slug: 'industry-guides', title: 'Credit guides for different businesses', description: 'See practical credit-sale advice for wholesalers, pharmacies, building suppliers, farmers and other Nigerian traders.' },
	'Business growth': { slug: 'business-growth', title: 'Business growth guides', description: 'Learn how clear credit records, staff routines and customer payment history can help a growing business stay in control.' }
};

export function categoryForSlug(slug: string) {
	return articleCategories.find(category => articleCategoryDetails[category].slug === slug);
}

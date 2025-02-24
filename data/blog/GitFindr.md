This is a  pretty chill overview of me trying to engineer the Google of Github...no prerequisites needed and hopefully it will be a fun read for everyone!

 Ever felt like Github search just doesn't cut it? Ever saw something you built that had the potential to be massive get buried under thousands of repos such that you barely get any traction from github itself? Enter Gitfindr, a Crowdsourced Search Engine build on Go prioritizing project discovery and deliver results that actually matter, ensuring no idea goes unnoticed.

The biggest problem with Github Search? It indexes its repos based on their Title, Description of the Repository and some other metadata like tags and languages used. This is cool and all for searching when I know exactly what I am looking for. But when you are casually searching for something like "AI tool to run commands on terminal Gemini", although there are a lot of cool projects like that of Github, the search returns...nothing! Not even a related section. Which sucks!

To fix this, the engine needs more context, i.e. more keywords! Now tell me, what is the one place that has many keywords, basically has more info about the repo and is present in most repos? Oh YES-THE README! So that is what we did, we indexed the readme along with the name, title and description etc so that our search engine has a better context of what to look for displaying the results.


Cool. Now that we have all these repos, how do we actually search through them?

Let’s play a game. Imagine you have **all of Shakespeare’s works** (yep, spelled it right on the first try), and I give you a challenge:

💡 _I’ll name a character, and your job is to instantly tell me all the plays they appear in—before anyone else does. If you’re the fastest, you win an internship!_

And guess what? You even get **prep time** before the game starts.

But how do you prepare? You could go through each play one by one and memorize everything manually, but that’s slow and inefficient. Instead, you create a **system** that lets you answer instantly when the challenge begins.

Without preparation, you’d have to **search through each play manually** every time I ask about a character:

|Play Name|Characters in the Play|
|---|---|
|_The Merchant of Venice_|Antonio, Shylock, Portia, Bassanio|
|_Much Ado About Nothing_|Benedick, Beatrice, Don Pedro|
|_Twelfth Night_|Viola, Sebastian, Malvolio, Antonio|

Now, if I ask: _"Which plays feature Antonio?"_, you’d have to **scan every row** to find him—a slow process, especially if you had thousands of plays.

Instead of scrambling through the data **after** the game starts, you prepare by reorganizing it beforehand:

🔄 **You flip the structure from**  
📜 **Play → List of Characters**  
🔁 **to**  
🎭 **Character → List of Plays They Appear In**

|Character|Plays They Appear In|
|---|---|
|Antonio|_The Merchant of Venice_, _Twelfth Night_|
|Shylock|_The Merchant of Venice_|
|Viola|_Twelfth Night_|

Now, when I ask for _Antonio_, instead of searching through **all** plays, you just do **one lookup**—instantly returning the answer.

This is exactly how **Gitfindr** searches through repositories. Instead of scanning repos one by one for keywords, we **preprocess everything into an efficient index**, allowing for **O(1) lookups** rather than slow searches.

We have a repo and to index it, we take its readme file, process it for the keywords, and then add them to the inverted index where we map the words -> repository

But there are still some issues! Synonyms. How do we handle synonyms? Someone might search for "Rust Terminal Apps" and get nothing, even though the exact thing they were looking for was under "Rust CMD Apps." Annoying, right?

The problem is that traditional search engines treat words as-is. They don’t realize that **"terminal" and "cmd" are the same thing** in this context. If the exact words don’t match, you’re out of luck. That’s a **big** issue if we want to make project discovery better than GitHub’s search.

So, here’s what I did. And no, this was **definitely** not a design disaster. (And that, my friends, is an example of foreshadowing.) 
I decided to group all synonyms into what I called a **"Synonym Cluster."** For example, _Hi_ and _Hello_ are synonyms, so they get assigned **the same index, say 1**. Meanwhile, _Bad_ is a new word, so it gets **index 2**, and so on.

The process was simple: when encountering a word, I’d check if it was already in my synonym database. If not, I’d fetch its synonyms—either through APIs or LLMs—and assign it a new **Synonym ID**.

Think of it like this: there’s _Bassanio_ in Shakespeare, but maybe he’s also referred to as _Bassino_ somewhere. Instead of storing both names separately in the database, we **just store an integer ID** that represents him. This saves space and makes searching much faster since we’re always looking up a **single ID instead of multiple variations** of the same word.

So now, when someone searches for _Bassanio_, we don’t just search for that exact string. Instead, we **convert it to its Synonym ID first** and use that to search—ensuring all related terms are matched effortlessly.

But then I realized—one word could belong to **multiple synonym clusters!** A single term could have different meanings depending on the context.

**Easy fix!** Instead of mapping words as `string -> int`, I switched to `string -> array of int`. Now, a word isn’t locked into just **one** cluster—it can belong to **several**.

This way, synonyms are accounted for... and it would _totally_ not make the search results overly vague. (_Foreshadowing intensifies._)
But we'll worry about the _results_ later! First, we need to keep track of **how often** each term appears in our database. This is important because not all words carry the same weight—some are way more common than others.

So, instead of just storing which repos contain a word, we'll also **track its frequency** to improve ranking. Our updated structure looks something like this:

| CharacterID | Appears In                                        |
| ----------- | ------------------------------------------------- |
| 1           | _The Merchant of Venice (2)_, _Twelfth Night (1)_ |
| 2           | _The Merchant of Venice (1)_                      |
| 3           | _Twelfth Night (1)_                               |

---

Now, with all this groundwork, we’re finally at the **most crucial stage**—ranking the results!

For this, we’ll use **BM25**, a statistical function that assigns each document a relevance score based on a query. Simply put, if you search for **"Apples and Bananas"**, BM25 will calculate a score for **"Apples"**, a score for **"Bananas"**, and then combine them to determine which documents are the **most relevant**. The higher the score, the better the match!

$${ BM25(D, Q) = \sum_{i=1}^{|Q|} IDF(q_i) \cdot \frac{f(q_i, D) \cdot (K+1)}{f(q_i, D) + K \cdot (1 - b + b \cdot \frac{|D|}{avgD})}  }$$


Now, let's break down the first half of the formula. The **f(q,D)** part? That’s just **"How many times does the term q appear in this specific repo D?"** Simple.

But then, we’ve got **k1k_1k1​ and bbb**—these are constants that fine-tune the score. And then there's this sneaky **$$\frac{|D|}{\text{avg}(|D|)}$$** term, which does something really important:

💡 **It penalizes long documents!**

Think about it—**longer repos** (especially those with **massive README files**) are **way** more likely to contain the term you're searching for. But just because a term appears a lot **doesn’t mean it’s actually the best match**. This part of the formula **keeps long repos in check**, making sure they don’t just dominate the rankings **for no good reason**.

Without this, a repo with a **super long README** could unfairly rank at the top just because it happens to contain the term a lot—even if it’s not actually that relevant. And **that’s not what we want.**

This is what makes BM25 **so much better** than just counting words—it **balances** relevance **instead of just boosting the biggest repos**.  
Now, the first part controls how much a term matters in a single repo, but what about across **all repos**? That’s where IDF comes in.
In this formula, the second part,  **Inverse Document Frequency (IDF)** is the part that makes sure we’re actually ranking results **intelligently**. Here’s the deal:

$$\text{IDF}(t) = \log \frac{N - N(t) + 0.5}{N(t) + 0.5}$$

Where:

- N is the total number of repositories.
- N(t) is the number of repositories where the term t appears.

Now, why does this matter? Because without it, our search engine would be absolutely **clueless** about what’s important. Let’s say you search for **"AGI Implementation"**—without IDF, the engine might go **insane over "Implementation"** just because it appears in a ton of repos. Meanwhile, **"AGI"**, the actually rare and valuable term, might get buried.

IDF **fixes this** by saying:  
_"Hey, if a word appears in almost every repo, it's probably not that special. But if it's rare? Ooooh, that's important!"_

Basically, IDF **tells the engine to chill on common words and focus on what actually matters**. Without it, searching would be a mess, and you'd end up with garbage results.


Hold on a second! I realized that the standard BM25 formula wasn’t quite right for our use case, so I had to tweak it. FIrst of all, not all words are equal. Descriptions, names and topics should be given more importance than the readme. So, I came across this paper on Wikipedia, and the author suggested increasing the frequency of those words, like count the title 5 times, the desc 3 times instead of 1 and so on. Also modify that k constant accordingly, and it did work pretty well.

Robertson, Stephen; Zaragoza, Hugo; Taylor, Michael (2004-11-13). ["Simple BM25 extension to multiple weighted fields"](https://doi.org/10.1145/1031171.1031181). _Proceedings of the thirteenth ACM international conference on Information and knowledge management_. CIKM '04. New York, NY, USA: Association for Computing Machinery. pp. 42–49. [doi](https://en.wikipedia.org/wiki/Doi_\(identifier\) "Doi (identifier)"):[10.1145/1031171.1031181](https://doi.org/10.1145%2F1031171.1031181). [ISBN](https://en.wikipedia.org/wiki/ISBN_\(identifier\) "ISBN (identifier)") [978-1-58113-874-0](https://en.wikipedia.org/wiki/Special:BookSources/978-1-58113-874-0 "Special:BookSources/978-1-58113-874-0"). [S2CID](https://en.wikipedia.org/wiki/S2CID_\(identifier\) "S2CID (identifier)") [16628332](https://api.semanticscholar.org/CorpusID:16628332).

But wait—ranking purely by text isn’t enough. Popularity matters too! Repos with more stars and forks are usually more relevant, so they deserve a say in the final ranking. So, I added documentwise formula that gets multiplied to the BM25 for each document. 

  $${ alt = (1 + \sum_{i} \alpha_i \log (1 + x_i)) }$$

Here, `x` represents a parameter like stars or forks. But notice the `log`? That’s crucial—if we kept it linear, a random repo with **50,000 stars** would crush a solid one with **100 stars**. The `log` levels the playing field, making sure popularity helps but doesn’t completely dominate. And `α`? That’s just a tuning constant—adjusting how much weight we actually give to these popularity signals.


So Now the code would simply pick the Query terms one by one, and see which all repos it is present in from the inverted index, and assign a score to each and the ones with the better score rise up.  

So, I indexed the repos and got the first set of results... and they were **disappointing, to say the least**. I mean, the system _kinda_ worked, but there was a major issue—my synonym sources were **too aggressive**.

For example, they treated **"game"** as a synonym for both **chess** and **sudoku**, which meant the search engine thought they were the _same thing_! Suddenly, chess-related repos were showing up for sudoku queries, and vice versa. It was a **disaster**.

At this point, I was stuck. I couldn't just **rewrite the entire English vocabulary**—that's obviously not an option. I also couldn't just **remove synonyms entirely** since they were a **core feature** of the project. That was the lowest point...

**But then it hit me!**

Then it hit me—I could **trade some efficiency for accuracy!**

Instead of using just **one** inverted index, I could use **two**. Yeah, that might sound dumb at first, but hear me out:

- One index maps **synonym clusters** to the repos they appear in.
- The other is a standard inverted index that maps **exact terms** to repos—just like you would have without synonyms.

Now, instead of relying _only_ on synonyms or _only_ on exact matches, I could **combine both scores**!

I’d compute **BM25 separately** for both tables and then add them up, giving **more weight** to the exact-term match. This way, if the synonyms help, they contribute—but they don’t **overpower** the search results and mess everything up.

Simple, effective, and no need to rewrite the English language. **Problem solved.**

Now the searches became far more accurate and reliable! No need to rewrite the whole damn codebase  after all these days! 

### Deployment 
### 🚀 **Deployment Plan**

In the next few days, I’ll be deploying this system so that **anyone can use it.** Since this is my first time deploying a full-fledged backend, I’m taking the time to learn **proper deployment strategies** to ensure stability, security, and performance.

The initial deployment will likely include:

- **Backend:** Running on a **cloud server (AWS, Hetzner, or DigitalOcean)** with a lightweight API layer to serve search queries.
- **Database:** PostgreSQL for structured repo metadata, plus **a custom inverted index** for BM25.
- **Indexing Jobs:** A background worker (probably using **Celery or Redis queues**) to periodically reindex repositories.
- **Caching:** **Redis** to speed up frequent queries and avoid recomputation.

The goal is to **get it working first**, with a focus on **usability and reliability**. Once the core system is stable, I'll shift focus to **optimizing for speed** and **handling higher loads** if necessary.

